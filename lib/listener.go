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

// RegistrationLocks ensure the user can't complete registration until they've finished the reg process.
type RegistrationLocks struct {
	sync.Mutex
	Cap  bool
	Nick bool
	User bool
	Pass bool
}

// Set sets the given registration lock.
func (rl *RegistrationLocks) Set(lockName string, val bool) {
	rl.Lock()
	defer rl.Unlock()

	switch lockName {
	case "cap":
		rl.Cap = val
	case "nick":
		rl.Nick = val
	case "user":
		rl.User = val
	case "pass":
		rl.Pass = val
	}
}

// Completed returns true if all of our registration locks have been completed.
func (rl *RegistrationLocks) Completed() bool {
	rl.Lock()
	completed := rl.Cap && rl.Pass && rl.Nick && rl.User
	rl.Unlock()
	return completed
}

// Listener is a listener for a client connected directly to us.
type Listener struct {
	Socket Socket

	Manager          *Manager
	ConnectTime      time.Time
	Caps             map[string]string
	ExtraISupports   map[string]string
	TagsEnabled      bool
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
		Caps:           make(map[string]string),
		ExtraISupports: make(map[string]string),
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

func (listener *Listener) IsCapEnabled(cap string) bool {
	_, enabled := listener.Caps[cap]
	return enabled
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

		listener.Manager.Bus.Dispatch(HookStateSentName, &HookStateSent{
			Listener: listener,
			Server:   listener.ServerConnection,
		})
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

func (listener *Listener) SendExtraISupports() {
	params := []string{listener.ClientNick}

	for token, val := range listener.ExtraISupports {
		if val != "" {
			params = append(params, fmt.Sprintf("%s=%s", token, val))
		} else {
			params = append(params, token)
		}
	}

	params = append(params, "are supported by this server")

	isupportMessage := ircmsg.MakeMessage(nil, listener.Manager.Source, "005", params...)
	listener.SendMessage(&isupportMessage)
}

// SendNilConnect sends a connection init (001+ERR_NOMOTD) to the listener when they are not connected to a server.
func (listener *Listener) SendNilConnect() {
	listener.Send(nil, listener.Source, "001", listener.ClientNick, "- Welcome to GoshuBNC -")
	listener.SendExtraISupports()
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
	if listener.ServerConnection != nil {
		listener.ServerConnection.RemoveListener(listener)
	}
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

	shouldHalt := Capabilities.MessageFromClient(listener, &msg)
	if shouldHalt {
		return
	}

	command, commandExists := ClientCommands[strings.ToUpper(msg.Command)]
	if commandExists {
		shouldHalt := command.Run(listener, msg)
		if shouldHalt {
			return
		}
	}

	// Forward the data
	if listener.Registered && listener.ServerConnection != nil {
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

// SendMessage sends an IrcMessage to the user
func (listener *Listener) SendMessage(msg *ircmsg.IrcMessage) error {
	return listener.Send(&msg.Tags, msg.Prefix, msg.Command, msg.Params...)
}

// Send sends an IRC line to the user.
func (listener *Listener) Send(tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) error {
	var message ircmsg.IrcMessage
	if listener.TagsEnabled {
		message = ircmsg.MakeMessage(tags, prefix, command, params...)
	} else {
		message = ircmsg.MakeMessage(nil, prefix, command, params...)
	}

	shouldHalt := Capabilities.MessageToClient(listener, &message)
	if shouldHalt {
		return nil
	}

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

	listener.SendLine(line)
	return nil
}

// SendLine sends a raw string line to the user.
func (listener *Listener) SendLine(line string) {
	listener.Socket.WriteLine(line)
}

// SendStatus sends a status PRIVMSG to the user.
func (listener *Listener) SendStatus(line string) {
	listener.Send(nil, listener.Manager.StatusSource, "PRIVMSG", listener.ClientNick, line)
}
