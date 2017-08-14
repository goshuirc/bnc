package ircclient

import (
	"github.com/goshuirc/irc-go/ircmsg"
)

func loadServerCommands() {
	ServerCommands["NICK"] = ServerCommand{
		minParams: 1,
		handler: func(client *Client, msg *ircmsg.IrcMessage) {
			client.Nick = msg.Params[0]
		},
	}
}
