// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/goshuirc/eventmgr"
	"github.com/goshuirc/irc-go/client"
	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/goshuirc/irc-go/ircmsg"
)

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name      string
	User      *User
	Connected bool

	Nickname   string
	FbNickname string
	Username   string
	Realname   string
	Channels   map[string]ServerConnectionChannel

	receiveLines  chan *string
	ReceiveEvents chan Message

	storingConnectMessages bool
	connectMessages        []ircmsg.IrcMessage
	currentServer          *gircclient.ServerConnection
	Listeners              []*Listener

	Password  string
	Addresses []ServerConnectionAddress
}

func NewServerConnection() *ServerConnection {
	return &ServerConnection{
		storingConnectMessages: true,
		receiveLines:           make(chan *string),
		ReceiveEvents:          make(chan Message),
	}
}

type ServerConnectionAddress struct {
	Host      string
	Port      int
	UseTLS    bool `json:"use-tls"`
	VerifyTLS bool `json:"verify-tls"`
}

type ServerConnectionAddresses []ServerConnectionAddress

type ServerConnectionChannel struct {
	Name   string
	Key    string
	UseKey bool `json:"use-key"`
}

type ServerConnectionChannels []ServerConnectionChannel

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
		listener.Send(nil, listener.Manager.StatusSource, "PRIVMSG", "Disconnected from server")
	}
}

func (sc *ServerConnection) rawToListeners(event string, info eventmgr.InfoMap) {
	line := info["data"].(string)
	msg, _ := ircmsg.ParseLine(line)

	hook := &HookIrcRaw{
		FromServer: true,
		User:       sc.User,
		Server:     sc,
		Raw:        line,
		Message:    msg,
	}
	sc.User.Manager.Bus.Dispatch(HookIrcRawName, hook)
	if hook.Halt {
		return
	}

	for _, listener := range sc.Listeners {
		if listener.Registered {
			listener.SendLine(line)
		}
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

	// dump reg
	for _, message := range sc.connectMessages {
		message.Params[0] = listener.ClientNick
		listener.Send(&message.Tags, message.Prefix, message.Command, message.Params...)
	}

	// change nick if user has a different one set
	if listener.ClientNick != sc.currentServer.Nick {
		listener.Send(nil, listener.ClientNick, "NICK", sc.currentServer.Nick)
		listener.ClientNick = sc.currentServer.Nick
	}
}

func (sc *ServerConnection) DumpChannels(listener *Listener) {
	for channel := range sc.Channels {
		//TODO(dan): add channel keys and enabled/disable bool here
		listener.Send(nil, sc.currentServer.Nick, "JOIN", channel)
		sc.currentServer.Send(nil, "", "NAMES", channel)
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
				continue
			}
			server.ProcessIncomingLine(*line)
		case msg = <-sc.ReceiveEvents:
			if msg.Type == AddListenerMT {
				listener := msg.Info[ListenerIK].(*Listener)
				sc.Listeners = append(sc.Listeners, listener)
				listener.ServerConnection = sc

				// registration blocks on the listener being added, continue if we should
				listener.regLocks["LISTENER"] = true
				listener.tryRegistration()
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
	server.RegisterEvent("in", "raw", sc.rawToListeners, 0)
	server.RegisterEvent("out", "server disconnected", sc.disconnectHandler, 0)
	server.RegisterEvent("in", "JOIN", sc.handleJoin, 0)
	server.RegisterEvent("in", "raw", rawHandler, 0)
	server.RegisterEvent("out", "raw", rawHandler, 0)

	for _, channel := range sc.Channels {
		server.JoinChannel(channel.Name, channel.Key, channel.UseKey)
	}

	var err error
	for _, address := range sc.Addresses {
		fullAddress := net.JoinHostPort(address.Host, strconv.Itoa(address.Port))

		var tlsConfig tls.Config
		if !address.VerifyTLS {
			tlsConfig.InsecureSkipVerify = true
		}

		err = server.Connect(fullAddress, address.UseTLS, &tlsConfig)
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

func (sc *ServerConnection) handleJoin(event string, info eventmgr.InfoMap) {
	params := info["params"].([]string)
	if len(params) < 1 {
		// invalid JOIN message
		return
	}

	var name, key string
	var useKey bool
	name = params[0]
	if 1 < len(params) && 0 < len(params[1]) {
		key = params[1]
		useKey = true
	}

	//TODO(dan): Store the new channel in the datastore
	//TODO(dan): On PARTs, remove the channel from the datastore as well
	log.Println("adding channel", name)
	sc.Channels[name] = ServerConnectionChannel{
		Name:   name,
		Key:    key,
		UseKey: useKey,
	}

}
