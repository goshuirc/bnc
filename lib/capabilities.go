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
	CapExtendedJoin(&Capabilities)
	CapAccountNotify(&Capabilities)
	CapAccountTag(&Capabilities)
	CapInviteNotify(&Capabilities)
	CapUserhostInNames(&Capabilities)
	CapBatch(&Capabilities)
}

// SupportedString returns a list ready to send to the client of all our CAPs
func (caps *CapManager) SupportedString() string {
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

// FilterSupported filters the supported CAPs by the requested
func (caps *CapManager) FilterSupported(requested []string) map[string]string {
	matched := make(map[string]string)

	for _, cap := range requested {
		capVal, isAvailable := Capabilities.Supported[cap]
		if isAvailable {
			matched[cap] = capVal
		}
	}

	return matched
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

/**
 * CAP: server-time
 */
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

/**
 * CAP: extended-join
 */
func CapExtendedJoin(caps *CapManager) {
	name := "extended-join"
	caps.Supported[name] = ""

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			// extened-join adds a param between params 1 + 2. Strip it out if the client
			// doesn't support it.
			if message.Command == "JOIN" && !listener.IsCapEnabled(name) && len(message.Params) == 3 {
				message.Params = []string{message.Params[0], message.Params[1]}
			}

			return false
		},
	)
}

/**
 * CAP: account-notify
 */
func CapAccountNotify(caps *CapManager) {
	name := "account-notify"
	caps.Supported[name] = ""

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			if message.Command == "ACCOUNT" && !listener.IsCapEnabled(name) {
				return true
			}

			return false
		},
	)
}

/**
 * CAP: account-tag
 */
func CapAccountTag(caps *CapManager) {
	name := "account-tag"
	caps.Supported[name] = ""

	caps.FnsInitListener[name] = func(listener *Listener) {
		listener.TagsEnabled = true
	}

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			// If the client has not enabled account-tag, but we're about to send
			// an account tag, then remove it
			if !listener.IsCapEnabled(name) {
				_, exists := message.Tags["account"]
				if exists {
					delete(message.Tags, "account")
				}
			}

			return false
		},
	)
}

/**
 * CAP: invite-notify
 */
func CapInviteNotify(caps *CapManager) {
	name := "invite-notify"
	caps.Supported[name] = ""

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			if message.Command == "INVITE" && !listener.IsCapEnabled(name) {
				return true
			}

			return false
		},
	)
}

/**
 * CAP: userhost-in-names
 */
func CapUserhostInNames(caps *CapManager) {
	name := "userhost-in-names"
	caps.Supported[name] = ""

	caps.FnsMessageToClient = append(
		caps.FnsMessageToClient,
		func(listener *Listener, message *ircmsg.IrcMessage) bool {
			// If the client hasn't enabled this cap, make sure that all names entries
			// only consist of the nick and not a full mask.
			if message.Command == "353" && !listener.IsCapEnabled(name) {
				names := strings.Split(message.Params[3], " ")
				for idx, name := range names {
					nick, _, _ := explodeHostmask(name)
					names[idx] = nick
				}

				message.Params[3] = strings.Join(names, " ")
			}

			return false
		},
	)
}

/**
 * CAP: batch
 * Not used on it's own, but other commands such as CHATHISTORY make use of it
 */
func CapBatch(caps *CapManager) {
	caps.Supported["batch"] = ""
}

func explodeHostmask(mask string) (string, string, string) {
	nick := ""
	username := ""
	host := ""

	pos := 0

	pos = strings.Index(mask, "!")
	if pos > -1 {
		nick = mask[0:pos]
		mask = mask[pos+1:]
	} else {
		nick = mask
		mask = ""
	}

	pos = strings.Index(mask, "@")
	if pos > -1 {
		username = mask[0:pos]
		mask = mask[pos+1:]
		host = mask
	} else {
		username = mask
		host = ""
	}

	return nick, username, host
}
