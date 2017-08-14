package ircclient

import "github.com/goshuirc/irc-go/ircmsg"

func (client *Client) handleCommonCommands() {
	// TODO: Move these commands into its own file. handle them in the same way we handle commands
	// from the client
	// TODO: Store ackd CAPs
	client.HandleCommand("NICK", func(message *ircmsg.IrcMessage) {
		if len(message.Params) >= 1 {
			client.Nick = message.Params[0]
		}
	})
}
