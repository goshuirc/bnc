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
	Bouncer     *Bouncer
	ConnectTime time.Time
	socket      Socket
	ClientNick  string
	Registered  bool
}

// NewListener creates a new Listener.
func NewListener(b *Bouncer, conn net.Conn) {
	now := time.Now()
	listener := &Listener{
		Bouncer:     b,
		ConnectTime: now,
		socket:      NewSocket(conn),
	}
	go listener.Run()
}

// Run starts and runs the listener.
func (listener *Listener) Run() {
	var errConn error
	var exiting bool
	var line string

	for {
		line, errConn = listener.socket.Read()
		if errConn != nil {
			break
		}
		exiting = listener.ProcessLine(line)
		if exiting {
			break
		}
	}
	listener.Send(nil, "", "ERROR", "Closing connection")
	listener.socket.Close()
}

// Send sends an IRC line to the listener.
func (listener *Listener) Send(tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) error {
	ircmsg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := ircmsg.Line()
	if err != nil {
		return err
	}
	return listener.socket.Write(line)
}

// ProcessLine splits and handles the given command line.
// Returns true if client is exiting (sent a QUIT command, etc).
func (listener *Listener) ProcessLine(line string) bool {
	msg, err := ircmsg.ParseLine(line)
	if err != nil {
		listener.Send(nil, "", "ERROR", "Your client sent a malformed line")
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
