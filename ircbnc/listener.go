// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"fmt"
	"net"
	"time"

	"github.com/DanielOaks/girc-go/ircmsg"
)

// Listener is a listener for a client connected directly to us.
type Listener struct {
	SocketReactor

	Bouncer     *Bouncer
	ConnectTime time.Time
	ClientNick  string
	Source      string
	Registered  bool
	regLocks    map[string]bool

	User             *User
	ServerConnection *ServerConnection
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
	listener.SocketReactor = NewSocketReactor(conn, listener.processIncomingLine)
	listener.Start()
}

// DumpRegistration dumps the registration numerics/replies to the listener.
func (listener *Listener) DumpRegistration() {
	sc := listener.ServerConnection
	if sc == nil {
		listener.Send(nil, listener.Source, "001", listener.ClientNick, "- Welcome to gIRCbnc -")
		listener.Send(nil, listener.Source, "422", listener.ClientNick, "MOTD File is missing")
		listener.Send(nil, listener.Bouncer.StatusSource, "NOTICE", listener.ClientNick, "You are not connected to any specific network")
		listener.Send(nil, listener.Bouncer.StatusSource, "NOTICE", listener.ClientNick, fmt.Sprintf("If you want to connect to a network, connect with the server password %s/<network>:<password>", "<username>"))
	} else {
		sc.DumpRegistration(listener)
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
	//TODO(dan): This is an error+disconnect purely for reasons of testing.
	// Later it may be downgraded to not-that-bad.
	listener.Send(nil, "", "ERROR", fmt.Sprintf("Your client sent a command that could not be parsed [%s]", msg.Command))
	return true
}
