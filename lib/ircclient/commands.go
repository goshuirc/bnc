// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

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
	handler   func(client *Client, msg *ircmsg.IrcMessage) bool
	minParams int
}

// Run runs this command with the given listener/message.
func (cmd *ServerCommand) Run(client *Client, msg *ircmsg.IrcMessage) bool {
	if len(msg.Params) < cmd.minParams {
		log.Println("Not enough parameters sent from the server: " + msg.SourceLine)
		return false
	}
	return cmd.handler(client, msg)
}
