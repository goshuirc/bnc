// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import "github.com/goshuirc/irc-go/ircmsg"

// Command represents a command accepted on a listener.
type Command struct {
	handler      func(listener *Listener, msg ircmsg.IrcMessage) bool
	usablePreReg bool
	minParams    int
}

// Run runs this command with the given listener/message.
func (cmd *Command) Run(listener *Listener, msg ircmsg.IrcMessage) bool {
	if !listener.Registered && !cmd.usablePreReg {
		// command silently ignored
		return false
	}
	if len(msg.Params) < cmd.minParams {
		listener.Send(nil, "", "461", listener.ClientNick, msg.Command, "Not enough parameters")
		return false
	}
	exiting := cmd.handler(listener, msg)

	// after each command, see if we can send registration to the listener
	if !listener.Registered {
		isRegistered := true
		for _, fulfilled := range listener.regLocks {
			if !fulfilled {
				isRegistered = false
				break
			}
		}
		if isRegistered {
			listener.DumpRegistration()
		}
	}

	return exiting
}

// Commands holds all commands executable by a client connected to a listener.
var Commands = map[string]Command{
	"NICK": Command{
		handler:      nickHandler,
		usablePreReg: true,
		minParams:    1,
	},
	"USER": Command{
		handler:      userHandler,
		usablePreReg: true,
		minParams:    4,
	},
	"PASS": Command{
		handler:      passHandler,
		usablePreReg: true,
		minParams:    1,
	},
	"CAP": Command{
		handler:      capHandler,
		usablePreReg: true,
		minParams:    1,
	},
}
