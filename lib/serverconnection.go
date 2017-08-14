// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/goshuirc/bnc/lib/ircclient"
	"github.com/goshuirc/irc-go/ircmsg"
)

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name      string
	User      *User
	Connected bool
	Enabled   bool

	Nickname   string
	FbNickname string
	Username   string
	Realname   string
	Channels   map[string]ServerConnectionChannel

	receiveLines  chan *string
	ReceiveEvents chan Message

	storingConnectMessages bool
	connectMessages        []ircmsg.IrcMessage
	Listeners              []*Listener

	Password  string
	Addresses []ServerConnectionAddress
	Foo       *ircclient.Client
}

func NewServerConnection() *ServerConnection {
	return &ServerConnection{
		storingConnectMessages: true,
		receiveLines:           make(chan *string),
		ReceiveEvents:          make(chan Message),
		Foo:                    ircclient.NewClient(),
	}
}

type ServerConnectionAddress struct {
	Host      string
	Port      int
	UseTLS    bool
	VerifyTLS bool
}

type ServerConnectionAddresses []ServerConnectionAddress

type ServerConnectionChannel struct {
	Name   string
	Key    string
	UseKey bool
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
func (sc *ServerConnection) disconnectHandler(message *ircmsg.IrcMessage) {
	for _, listener := range sc.Listeners {
		listener.Send(nil, listener.Manager.StatusSource, "PRIVMSG", "Disconnected from server")
	}
}

func (sc *ServerConnection) rawToListeners(message *ircmsg.IrcMessage) {
	hook := &HookIrcRaw{
		FromServer: true,
		User:       sc.User,
		Server:     sc,
		Raw:        message.SourceLine,
		Message:    *message,
	}
	sc.User.Manager.Bus.Dispatch(HookIrcRawName, hook)
	if hook.Halt {
		return
	}

	for _, listener := range sc.Listeners {
		if listener.Registered {
			listener.SendLine(message.SourceLine)
		}
	}
}

// connectLinesHandler extracts and stores the connection lines.
func (sc *ServerConnection) connectLinesHandler(message *ircmsg.IrcMessage) {
	if !sc.storingConnectMessages || message == nil {
		return
	}

	_, storeMessage := storedConnectLines[message.Command]
	if storeMessage {
		// fmt.Println("IN:", message)
		sc.connectMessages = append(sc.connectMessages, *message)
	}

	if message.Command == "376" || message.Command == "422" {
		sc.storingConnectMessages = false
	}
}

// DumpRegistration dumps the registration messages of this server to the given Listener.
func (sc *ServerConnection) DumpRegistration(listener *Listener) {
	// if server is not currently connected, just dump a nil connect
	if !sc.Foo.Connected {
		listener.SendNilConnect()
		return
	}

	// dump reg
	for _, message := range sc.connectMessages {
		message.Params[0] = listener.ClientNick
		listener.Send(&message.Tags, message.Prefix, message.Command, message.Params...)
	}

	// change nick if user has a different one set
	if listener.ClientNick != sc.Foo.Nick {
		listener.Send(nil, listener.ClientNick, "NICK", sc.Foo.Nick)
		listener.ClientNick = sc.Foo.Nick
	}
}

func (sc *ServerConnection) DumpChannels(listener *Listener) {
	for channel := range sc.Channels {
		//TODO(dan): add channel keys and enabled/disable bool here
		listener.Send(nil, sc.Foo.Nick, "JOIN", channel)
		sc.Foo.WriteLine("NAMES %s", channel)
	}
}

// ReceiveLoop runs a loop of receiving and dispatching new messages.
func (sc *ServerConnection) ReceiveLoop() {
	for {
		msg := <-sc.ReceiveEvents

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

// AddListener adds the given listener to this ServerConnection.
func (sc *ServerConnection) AddListener(listener *Listener) {
	message := NewMessage(AddListenerMT, NoMV)
	message.Info[ListenerIK] = listener
	sc.ReceiveEvents <- message
}

// Start opens and starts connecting to the server.
func (sc *ServerConnection) Start() {
	name := fmt.Sprintf("%s %s", sc.User.ID, sc.Name)

	sc.Foo.Nick = sc.Nickname
	sc.Foo.Username = sc.Username
	sc.Foo.Realname = sc.Realname
	sc.Foo.Password = sc.Password

	sc.Foo.HandleCommand("ALL", sc.connectLinesHandler)
	sc.Foo.HandleCommand("ALL", sc.rawToListeners)
	sc.Foo.HandleCommand("CLOSED", sc.disconnectHandler)
	sc.Foo.HandleCommand("JOIN", sc.handleJoin)

	for _, channel := range sc.Channels {
		sc.Foo.JoinChannel(channel.Name, channel.Key)
	}

	var err error
	for _, address := range sc.Addresses {
		sc.Foo.Host = address.Host
		sc.Foo.Port = address.Port
		sc.Foo.TLS = address.UseTLS

		tlsConfig := &tls.Config{}
		if !address.VerifyTLS {
			tlsConfig.InsecureSkipVerify = true
		}
		sc.Foo.TLSConfig = tlsConfig

		err := sc.Foo.Connect()
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Println("ERROR: Could not connect to", name, err.Error())
		return
	}

	go sc.ReceiveLoop()
}

func (sc *ServerConnection) handleJoin(message *ircmsg.IrcMessage) {
	params := message.Params
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
