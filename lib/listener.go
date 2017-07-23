// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"fmt"
	"net"
	"time"

	"code.cloudfoundry.org/bytefmt"

	"log"

	"github.com/goshuirc/irc-go/ircmsg"
)

// Listener is a listener for a client connected directly to us.
type Listener struct {
	Socket Socket

	Bouncer     *Bouncer
	ConnectTime time.Time
	ClientNick  string
	Source      string
	Registered  bool
	regLocks    map[string]bool

	User             *User
	ServerConnection *ServerConnection
}

// RunSocketReader reads lines from the listener socket and dispatches them as appropriate.
func (listener *Listener) RunSocketReader() {
	for {
		line, err := listener.Socket.Read()
		if err != nil {
			break
		}
		listener.processIncomingLine(line)
	}
}

// NewListener creates a new Listener.
func NewListener(b *Bouncer, conn net.Conn) {
	now := time.Now()
	listener := &Listener{
		Bouncer:     b,
		ClientNick:  "*",
		ConnectTime: now,
		Source:      b.Source,
		regLocks: map[string]bool{
			"CAP":  true,
			"NICK": false,
			"USER": false,
		},
	}

	maxSendQBytes, _ := bytefmt.ToBytes("32k")
	listener.Socket = NewSocket(conn, maxSendQBytes)
	go listener.Socket.RunSocketWriter()
	go listener.RunSocketReader()
}

// tryRegistration dumps the registration blob and all if it hasn't been sent already.
func (listener *Listener) tryRegistration() {
	if listener.Registered {
		return
	}
	isRegistered := true
	for _, fulfilled := range listener.regLocks {
		if !fulfilled {
			isRegistered = false
			break
		}
	}
	if isRegistered {
		listener.DumpRegistration()
		listener.Registered = true
		listener.DumpChannels()
	}
}

// DumpRegistration dumps the registration numerics/replies to the listener.
func (listener *Listener) DumpRegistration() {
	sc := listener.ServerConnection
	if sc == nil {
		listener.SendNilConnect()
	} else {
		sc.DumpRegistration(listener)
	}
}

// SendNilConnect sends a connection init (001+ERR_NOMOTD) to the listener when they are not connected to a server.
func (listener *Listener) SendNilConnect() {
	listener.Send(nil, listener.Source, "001", listener.ClientNick, "- Welcome to GoshuBNC -")
	listener.Send(nil, listener.Source, "422", listener.ClientNick, "MOTD File is missing")
	listener.Send(nil, listener.Bouncer.StatusSource, "NOTICE", listener.ClientNick, "You are not connected to any specific network")
	listener.Send(nil, listener.Bouncer.StatusSource, "NOTICE", listener.ClientNick, fmt.Sprintf("If you want to connect to a network, connect with the server password %s/<network>:<password>", "<username>"))
}

// DumpChannels dumps the active channels to the listener.
func (listener *Listener) DumpChannels() {
	if listener.ServerConnection != nil {
		listener.ServerConnection.DumpChannels(listener)
	}
}

// processIncomingLine splits and handles the given command line.
// Returns true if client is exiting (sent a QUIT command, etc).
func (listener *Listener) processIncomingLine(line string) bool {
	msg, err := ircmsg.ParseLine(line)
	if err != nil {
		listener.Send(nil, "", "ERROR", "Your client sent a malformed line")
		return true
	}

	command, canBeParsed := Commands[msg.Command]

	if canBeParsed {
		return command.Run(listener, msg)
	}

	if listener.Registered {
		err := listener.ServerConnection.currentServer.Send(&msg.Tags, msg.Prefix, msg.Command, msg.Params...)
		if err != nil {
			log.Println(err.Error())
		}
	}

	return false

	//TODO(dan): This is an error+disconnect purely for reasons of testing.
	// Later it may be downgraded to not-that-bad.
	// listener.Send(nil, "", "ERROR", fmt.Sprintf("Your client sent a command that could not be parsed [%s]", msg.Command))
	// return true
}

// Send sends an IRC line to the listener.
func (listener *Listener) Send(tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) error {
	// send out the message
	message := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := message.Line()
	if err != nil {
		// try not to fail quietly - especially useful when running tests, as a note to dig deeper
		// log.Println("Error assembling message:")
		// spew.Dump(message)
		// debug.PrintStack()

		message = ircmsg.MakeMessage(nil, "", ERR_UNKNOWNERROR, "*", "Error assembling message for sending")
		line, _ := message.Line()
		listener.Socket.Write(line)
		return err
	}

	listener.Socket.Write(line)
	return nil
}

// SendLine sends a raw string line to the listener
func (listener *Listener) SendLine(line string) {
	listener.Socket.WriteLine(line)
}
