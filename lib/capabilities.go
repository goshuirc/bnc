package ircbnc

import (
	"strings"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
)

type CapManager struct {
	Supported            map[string]string
	FnsInitListener      map[string]func(*Listener)
	FnsMessageToClient   []func(*Listener, *ircmsg.IrcMessage) bool
	FnsMessageFromClient []func(*Listener, *ircmsg.IrcMessage) bool
}

var Capabilities CapManager

func init() {
	Capabilities = CapManager{
		Supported:       make(map[string]string),
		FnsInitListener: make(map[string]func(*Listener)),
	}

	CapAwayNotify(&Capabilities)
	CapServerTime(&Capabilities)
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
func (caps *CapManager) InitCapOnListener(listener *Listener, cap string) {
	fn, exists := caps.FnsInitListener[cap]
	if exists {
		fn(listener)
	}
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

func CapServerTime(caps *CapManager) {
	name := "server-time"
	caps.Supported[name] = ""

	caps.FnsInitListener[name] = func(listener *Listener) {
		listener.TagsEnabled = true
	}

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			if !listener.IsCapEnabled(name) {
				return false
			}

			_, exists := message.Tags["time"]
			if !exists {
				message.Tags["time"] = ircmsg.TagValue{
					Value:    time.Now().UTC().Format(time.RFC3339),
					HasValue: true,
				}
			}

			return false
		},
	)
}
