package ircclient

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

func loadServerCommands() {
	ServerCommands[RPL_WELCOME] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.Lock()
			client.Nick = msg.Params[0]
			client.HasRegistered = true
			client.Unlock()
		},
	}

	ServerCommands[RPL_ISUPPORT] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
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
		},
	}

	ServerCommands[ERR_NICKNAMEINUSE] = ServerCommand{
		minParams: 0,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			if client.HasRegistered {
				return
			}

			// TODO: This should use the fallback nick set ont he client
			client.Lock()
			client.Nick = client.Nick + "_"
			client.Unlock()

			client.WriteLine("NICK %s", client.Nick)
		},
	}

	ServerCommands["NICK"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.Lock()
			client.Nick = msg.Params[0]
			client.Unlock()
		},
	}

	ServerCommands["PING"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.WriteLine("PONG :%s", msg.Params[0])
		},
	}

	ServerCommands["CAP"] = ServerCommand{
		minParams: 2,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
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
		},
	}
}

func getParam(msg *ircmsg.IrcMessage, idx int) string {
	if len(msg.Params)-1 < idx {
		return ""
	}

	return msg.Params[idx]
}
