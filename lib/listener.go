// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"

	"log"

	"github.com/goshuirc/bnc/lib/ircclient"
	"github.com/goshuirc/irc-go/ircmsg"
)

type RegistrationLocks struct {
	Lock sync.Mutex
	Cap  bool
	Nick bool
	User bool
	Pass bool
}

func (locks *RegistrationLocks) Set(lockName string, val bool) {
	locks.Lock.Lock()

	switch lockName {
	case "cap":
		locks.Cap = val
	case "nick":
		locks.Nick = val
	case "user":
		locks.User = val
	case "pass":
		locks.Pass = val
	}

	locks.Lock.Unlock()
}

func (locks *RegistrationLocks) Completed() bool {
	locks.Lock.Lock()
	completed := locks.Cap && locks.Pass && locks.Nick && locks.User
	locks.Lock.Unlock()
	return completed
}

// Listener is a listener for a client connected directly to us.
type Listener struct {
	Socket Socket

	Manager          *Manager
	ConnectTime      time.Time
	ClientNick       string
	Source           string
	Registered       bool
	regLocks         *RegistrationLocks
	User             *User
	ServerConnection *ServerConnection
}

// NewListener creates a new Listener.
func NewListener(m *Manager, conn net.Conn) {
	now := time.Now()
	listener := &Listener{
		Manager:     m,
		ClientNick:  "*",
		ConnectTime: now,
		Source:      m.Source,
		regLocks: &RegistrationLocks{
			Cap:  true,
			Nick: false,
			User: false,
			Pass: false,
		},
	}

	maxSendQBytes, _ := bytefmt.ToBytes("32k")
	listener.Socket = NewSocket(conn, maxSendQBytes)

	hook := &HookNewListener{
		Listener: listener,
	}
	m.Bus.Dispatch(HookNewListenerName, hook)
	if hook.Halt {
		listener.Socket.Close()
		return
	}

	go listener.Socket.RunSocketWriter()
	listener.RunSocketReader()
}

// tryRegistration dumps the registration blob and all if it hasn't been sent already.
func (listener *Listener) tryRegistration() {
	if listener.Registered {
		return
	}

	if listener.regLocks.Completed() {
		listener.DumpRegistration()
		listener.Registered = true
		listener.DumpChannels()
	}
}

// DumpRegistration dumps the registration numerics/replies to the listener.
func (listener *Listener) DumpRegistration() {
	if listener.ServerConnection == nil {
		listener.SendNilConnect()
	} else {
		listener.ServerConnection.DumpRegistration(listener)
	}
}

// SendNilConnect sends a connection init (001+ERR_NOMOTD) to the listener when they are not connected to a server.
func (listener *Listener) SendNilConnect() {
	listener.Send(nil, listener.Source, "001", listener.ClientNick, "- Welcome to GoshuBNC -")
	listener.Send(nil, listener.Source, "422", listener.ClientNick, "MOTD File is missing")
	listener.Send(nil, listener.Manager.StatusSource, "NOTICE", listener.ClientNick, "You are not connected to any specific network")
	listener.Send(nil, listener.Manager.StatusSource, "NOTICE", listener.ClientNick, fmt.Sprintf("If you want to connect to a network, connect with the server password %s/<network>:<password>", "<username>"))
}

// DumpChannels dumps the active channels to the listener.
func (listener *Listener) DumpChannels() {
	if listener.ServerConnection != nil {
		listener.ServerConnection.DumpChannels(listener)
	}
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

	listener.Manager.Bus.Dispatch(HookListenerCloseName, &HookListenerClose{
		Listener: listener,
	})
}

// processIncomingLine splits and handles the given command line.
// Returns true if client is exiting (sent a QUIT command, etc).
func (listener *Listener) processIncomingLine(line string) {
	// Don't let a single users error kill the entire bnc for everyone
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Recovered from %s\n%s", r, debug.Stack())
		}
	}()

	msg, parseLineErr := ircmsg.ParseLine(line)

	// Trigger the event if the line parsed or not just incase something else wants to
	// deal with them
	hook := &HookIrcRaw{
		FromClient: true,
		Listener:   listener,
		User:       listener.User,
		Server:     listener.ServerConnection,
		Raw:        line,
		Message:    msg,
	}
	listener.Manager.Bus.Dispatch(HookIrcRawName, hook)
	if hook.Halt {
		return
	}

	if parseLineErr != nil {
		listener.Send(nil, "", "ERROR", "Your client sent a malformed line")
		return
	}

	command, commandExists := ClientCommands[strings.ToUpper(msg.Command)]
	if commandExists {
		command.Run(listener, msg)
		return
	}

	if listener.Registered {
		line, _ := msg.Line()
		_, err := listener.ServerConnection.Foo.WriteLine(line)
		if err != nil {
			log.Println(err.Error())
		}
	}

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

		message = ircmsg.MakeMessage(nil, "", ircclient.ERR_UNKNOWNERROR, "*", "Error assembling message for sending")
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

func (listener *Listener) SendStatus(line string) {
	listener.Send(nil, listener.Manager.StatusSource, "PRIVMSG", listener.ClientNick, line)
}
