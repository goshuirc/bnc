// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"encoding/json"

	"github.com/goshuirc/eventmgr"
	"github.com/goshuirc/irc-go/client"
	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/tidwall/buntdb"
)

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name      string
	User      User
	Connected bool

	Nickname   string
	FbNickname string
	Username   string
	Realname   string
	Channels   map[string]string

	receiveLines  chan *string
	ReceiveEvents chan Message

	storingConnectMessages bool
	connectMessages        []ircmsg.IrcMessage
	currentServer          *gircclient.ServerConnection
	Listeners              []Listener

	Password  string
	Addresses []ServerConnectionAddress
}

// LoadServerConnection loads the given server connection from our database.
func LoadServerConnection(name string, user User, tx *buntdb.Tx) (*ServerConnection, error) {
	var sc ServerConnection
	sc.storingConnectMessages = true
	sc.receiveLines = make(chan *string)
	sc.ReceiveEvents = make(chan Message)
	sc.Name = name
	sc.User = user

	// load general info
	var scInfo ServerConnectionInfo
	scInfoString, err := tx.Get(fmt.Sprintf(KeyServerConnectionInfo, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc details from db): %s", err.Error())
	}

	err = json.Unmarshal([]byte(scInfoString), scInfo)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc details): %s", err.Error())
	}

	sc.Nickname = scInfo.Nickname
	sc.FbNickname = scInfo.NicknameFallback
	sc.Username = scInfo.Username
	sc.Realname = scInfo.Realname
	sc.Password = scInfo.ConnectPassword

	// set default values
	if sc.Nickname == "" {
		sc.Nickname = user.DefaultNick
	}
	if sc.FbNickname == "" {
		sc.FbNickname = user.DefaultFbNick
	}
	if sc.Username == "" {
		sc.Username = user.DefaultUser
	}
	if sc.Realname == "" {
		sc.Realname = user.DefaultReal
	}

	// load channels
	scChannelString, err := tx.Get(fmt.Sprintf(KeyServerConnectionChannels, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc channels from db): %s", err.Error())
	}

	var scChans ServerConnectionChannels
	err = json.Unmarshal([]byte(scChannelString), scChans)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc channels): %s", err.Error())
	}

	sc.Channels = make(map[string]string)
	for _, channel := range scChans {
		//TODO(dan): Store channel key and whether to use key here too, etc etc
		sc.Channels[channel.Name] = channel.Name
	}

	// load addresses
	scAddressesString, err := tx.Get(fmt.Sprintf(KeyServerConnectionAddresses, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc addresses from db): %s", err.Error())
	}

	var scAddresses ServerConnectionAddresses
	err = json.Unmarshal([]byte(scAddressesString), scAddresses)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc addresses): %s", err.Error())
	}

	// check port number and add addresses
	for _, address := range scAddresses {
		if address.Port < 1 || address.Port > 65535 {
			return nil, fmt.Errorf("Could not create new ServerConnection (port %d is not valid)", address.Port)
		}

		sc.Addresses = append(sc.Addresses, address)
	}

	return &sc, nil
}

//TODO(dan): Make all these use numeric names rather than numeric numbers
var storedConnectLines = map[string]bool{
	RPL_WELCOME:       true,
	RPL_YOURHOST:      true,
	RPL_CREATED:       true,
	RPL_MYINFO:        true,
	RPL_ISUPPORT:      true,
	"250":             true,
	RPL_LUSERCLIENT:   true,
	RPL_LUSEROP:       true,
	RPL_LUSERCHANNELS: true,
	RPL_LUSERME:       true,
	"265":             true,
	"266":             true,
	RPL_MOTD:          true,
	RPL_MOTDSTART:     true,
	RPL_ENDOFMOTD:     true,
	ERR_NOMOTD:        true,
}

// disconnectHandler extracts and stores .
func (sc *ServerConnection) disconnectHandler(event string, info eventmgr.InfoMap) {
	sc.currentServer = nil

	for _, listener := range sc.Listeners {
		listener.Send(nil, listener.Bouncer.StatusSource, "PRIVMSG", "Disconnected from server")
	}
}

// connectLinesHandler extracts and stores the connection lines.
func (sc *ServerConnection) connectLinesHandler(event string, info eventmgr.InfoMap) {
	if !sc.storingConnectMessages {
		return
	}

	line := info["data"].(string)
	message, err := ircmsg.ParseLine(line)
	if err != nil {
		return
	}

	_, storeMessage := storedConnectLines[message.Command]
	if storeMessage {
		// fmt.Println("IN:", message)
		sc.connectMessages = append(sc.connectMessages, message)
	}

	if message.Command == "376" || message.Command == "422" {
		sc.storingConnectMessages = false
	}
}

// DumpRegistration dumps the registration messages of this server to the given Listener.
func (sc *ServerConnection) DumpRegistration(listener *Listener) {
	// if server is not currently connected, just dump a nil connect
	if sc.currentServer == nil {
		listener.SendNilConnect()
		return
	}

	// change nick if user has a different one set
	//TODO(dan): If nick if diff. we may want to dump a NICK message, but maybe not.
	// If clients get nick from 001, it'll be fine.
	listener.ClientNick = sc.currentServer.Nick

	// dump reg
	for _, message := range sc.connectMessages {
		message.Params[0] = listener.ClientNick
		listener.Send(&message.Tags, message.Prefix, message.Command, message.Params...)
	}

}

// rawHandler prints raw messages to and from the server.
//TODO(dan): This is only VERY INITIAL, for use while we are debugging.
func rawHandler(event string, info eventmgr.InfoMap) {
	server := info["server"].(*gircclient.ServerConnection)
	direction := info["direction"].(string)
	line := info["data"].(string)

	var arrow string
	if direction == "in" {
		arrow = "<- "
	} else {
		arrow = " ->"
	}

	fmt.Println(server.Name, arrow, ircfmt.Escape(strings.Trim(line, "\r\n")))
}

func (sc *ServerConnection) lineReceiveLoop(server *gircclient.ServerConnection) {
	// wait for the connection to become available
	server.WaitForConnection()

	reader := bufio.NewReader(server.RawConnection)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			sc.receiveLines <- nil
			break
		}

		sc.receiveLines <- &line
	}

	server.Disconnect()
}

// ReceiveLoop runs a loop of receiving and dispatching new messages.
func (sc *ServerConnection) ReceiveLoop(server *gircclient.ServerConnection) {
	var msg Message
	var line *string
	for {
		select {
		case line = <-sc.receiveLines:
			if line == nil {
				break
			}
			server.ProcessIncomingLine(*line)
		case msg = <-sc.ReceiveEvents:
			if msg.Type == AddListenerMT {
				listener := msg.Info[ListenerIK].(*Listener)
				sc.Listeners = append(sc.Listeners, *listener)
				listener.ServerConnection = sc
			} else {
				log.Fatal("Got an event I cannot parse")
				fmt.Println(msg)
			}
		}
	}
}

// AddListener adds the given listener to this ServerConnection.
func (sc *ServerConnection) AddListener(listener *Listener) {
	message := NewMessage(AddListenerMT, NoMV)
	message.Info[ListenerIK] = listener
	sc.ReceiveEvents <- message
}

// Start opens and starts connecting to the server.
func (sc *ServerConnection) Start(reactor gircclient.Reactor) {
	name := fmt.Sprintf("%s %s", sc.User.ID, sc.Name)
	server := reactor.CreateServer(name)
	sc.currentServer = server

	server.InitialNick = sc.Nickname
	server.InitialUser = sc.Username
	server.InitialRealName = sc.Realname
	server.ConnectionPass = sc.Password
	server.FallbackNicks = append(server.FallbackNicks, sc.FbNickname)

	server.RegisterEvent("in", "raw", sc.connectLinesHandler, 0)
	server.RegisterEvent("out", "server disconnected", sc.disconnectHandler, 0)
	server.RegisterEvent("in", "raw", rawHandler, 0)
	server.RegisterEvent("out", "raw", rawHandler, 0)

	var err error
	for _, address := range sc.Addresses {
		fullAddress := net.JoinHostPort(address.Host, strconv.Itoa(address.Port))

		err = server.Connect(fullAddress, address.UseTLS, nil)
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Println("ERROR: Could not connect to", name, err.Error())
		return
	}

	go sc.lineReceiveLoop(server)
	go sc.ReceiveLoop(server)
}
