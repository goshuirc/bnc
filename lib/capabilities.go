package ircbnc

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

type CapManager struct {
	Supported            map[string]string
	FnsMessageToClient   []func(*Listener, *ircmsg.IrcMessage) bool
	FnsMessageFromClient []func(*Listener, *ircmsg.IrcMessage) bool
}

var Capabilities CapManager

func init() {
	Capabilities = CapManager{
		Supported: make(map[string]string),
	}
	CapAwayNotify(&Capabilities)
}

// AsString returns a list ready to send to the client of all our CAPs
func (caps *CapManager) AsString() string {
	capList := " "

	for cap, val := range caps.Supported {
		capList += cap
		if val != "" {
			capList += "=" + val
		}
		capList += " "
	}

	return strings.Trim(capList, " ")
}

// MessageToClient runs messages through any CAPs before being sent to the client
func (caps *CapManager) MessageToClient(listener *Listener, message *ircmsg.IrcMessage) bool {
	for _, fn := range caps.FnsMessageToClient {
		shouldHalt := fn(listener, message)
		if shouldHalt {
			return true
		}
	}

	return false
}

// MessageFromClient runs messages received from a client through any CAPs
func (caps *CapManager) MessageFromClient(listener *Listener, message *ircmsg.IrcMessage) bool {
	for _, fn := range caps.FnsMessageFromClient {
		shouldHalt := fn(listener, message)
		if shouldHalt {
			return true
		}
	}

	return false
}

/**
 * CAP: away-notify
 */
func CapAwayNotify(caps *CapManager) {
	name := "away-notify"
	caps.Supported[name] = ""

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			if message.Command == "AWAY" && !listener.IsCapEnabled(name) {
				return true
			}

			return false
		},
	)
}
