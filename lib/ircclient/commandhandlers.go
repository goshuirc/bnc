package ircclient

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

func loadServerCommands() {
	// WELCOME
	ServerCommands["001"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.Nick = msg.Params[0]
			client.HasRegistered = true
		},
	}

	// ISUPPORT
	ServerCommands["005"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
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

	// ERR_NICKNAMEINUSE
	ServerCommands["433"] = ServerCommand{
		minParams: 0,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			if client.HasRegistered {
				return
			}

			// TODO: This should use the fallback nick set ont he client
			client.Nick = client.Nick + "_"
			client.WriteLine("NICK %s", client.Nick)
		},
	}

	ServerCommands["NICK"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.Nick = msg.Params[0]
		},
	}

	ServerCommands["PING"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.WriteLine("PONG :%s", msg.Params[0])
		},
	}

}
