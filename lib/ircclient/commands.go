package ircclient

import (
	"log"

	"github.com/goshuirc/irc-go/ircmsg"
)

// ServerCommands holds all commands to be listened for from the server
var ServerCommands map[string]ServerCommand

func init() {
	ServerCommands = make(map[string]ServerCommand)
	loadServerCommands()
}

// ClientCommand represents a command accepted on a listener.
type ServerCommand struct {
	handler   func(client *Client, msg *ircmsg.IrcMessage)
	minParams int
}

// Run runs this command with the given listener/message.
func (cmd *ServerCommand) Run(client *Client, msg *ircmsg.IrcMessage) {
	if len(msg.Params) < cmd.minParams {
		log.Println("Not enough parameters sent from the server: " + msg.SourceLine)
	}
	cmd.handler(client, msg)
}
