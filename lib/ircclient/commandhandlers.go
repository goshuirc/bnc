// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package ircclient

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

func loadServerCommands() {
	ServerCommands[RPL_WELCOME] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			client.Lock()
			client.Nick = msg.Params[0]
			client.HasRegistered = true
			client.Unlock()

			return false
		},
	}

	ServerCommands[RPL_ISUPPORT] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			client.Lock()
			defer client.Unlock()

			supported := msg.Params[1 : len(msg.Params)-1]
			for _, item := range supported {
				parts := strings.SplitN(item, "=", 2)
				if len(parts) == 1 {
					client.Supported[parts[0]] = ""
				} else {
					client.Supported[parts[0]] = parts[1]
				}
			}

			return false
		},
	}

	ServerCommands[ERR_NICKNAMEINUSE] = ServerCommand{
		minParams: 0,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			if client.HasRegistered {
				return true
			}

			// TODO: This should use the fallback nick set ont he client
			client.Lock()
			client.Nick = client.Nick + "_"
			client.Unlock()

			client.WriteLine("NICK %s", client.Nick)

			return true
		},
	}

	ServerCommands["NICK"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			prefixNick, _, _ := SplitMask(msg.Prefix)

			// If our nick just changed, update ourselves
			if strings.ToLower(prefixNick) == strings.ToLower(client.Nick) {
				client.Lock()
				client.Nick = msg.Params[0]
				client.Unlock()
			}

			return false
		},
	}

	ServerCommands["PING"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			client.WriteLine("PONG :%s", msg.Params[0])
			return true
		},
	}

	ServerCommands["CAP"] = ServerCommand{
		minParams: 2,
		handler: func(client *Client, msg *ircmsg.IrcMessage) bool {
			command := msg.Params[1]

			if command == "LS" && !client.HasRegistered {
				capsRaw := ""
				isLastCapsLine := true

				// Multiline list
				if getParam(msg, 2) == "*" {
					capsRaw = getParam(msg, 3)
					isLastCapsLine = false
				} else {
					capsRaw = getParam(msg, 2)
				}

				for _, cap := range strings.Split(capsRaw, " ") {
					parts := strings.Split(cap, "=")
					k := strings.ToLower(parts[0])
					v := ""
					if len(parts) > 1 {
						v = parts[1]
					}

					client.Lock()
					client.Caps.Available[k] = v
					client.Unlock()
				}

				if isLastCapsLine {
					common := client.Caps.CommonCaps()
					if len(common) > 0 {
						client.WriteLine("CAP REQ :%s", strings.Join(common, " "))
					} else {
						client.WriteLine("CAP END")
					}
				}
			}

			if command == "ACK" {
				// TODO: Do the ACK responses also use * to denote a multiline response?
				capsRaw := getParam(msg, 2)

				for _, cap := range strings.Split(capsRaw, " ") {
					parts := strings.Split(cap, "=")
					k := strings.ToLower(parts[0])
					v := ""
					if len(parts) > 1 {
						v = parts[1]
					}

					client.Lock()
					client.Caps.Enabled[k] = v
					client.Unlock()
				}

				client.WriteLine("CAP END")
			}

			if command == "NAK" {
				// TODO: This
			}

			return true
		},
	}
}

func getParam(msg *ircmsg.IrcMessage, idx int) string {
	if len(msg.Params)-1 < idx {
		return ""
	}

	return msg.Params[idx]
}

func SplitMask(mask string) (string, string, string) {
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
