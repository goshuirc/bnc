package ircclient

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

type Client struct {
	Socket
	Nick             string
	Username         string
	Realname         string
	Password         string
	BindHost         string
	Caps             []string
	Supported        map[string]string
	HasRegistered    bool
	CommandListeners map[string][]func(*ircmsg.IrcMessage)
}

func NewClient() *Client {
	client := &Client{}
	go client.messageDispatcher()
	return client
}

func (client *Client) Connect() error {
	err := client.Socket.Connect()
	if err != nil {
		return err
	}

	client.WriteLine("%s 0 * :%s", client.Username, client.Realname)
	client.WriteLine("NICK %s", client.Nick)

	return nil
}

func (client *Client) HandleCommand(command string, fn func(*ircmsg.IrcMessage)) {
	command = strings.ToUpper(command)

	ar, _ := client.CommandListeners[command]
	if ar == nil {
		ar = make([]func(*ircmsg.IrcMessage), 0)
		client.CommandListeners[command] = ar
	}

	client.CommandListeners[command] = append(client.CommandListeners[command], fn)
}

func (client *Client) messageDispatcher() {
	var handlers []func(*ircmsg.IrcMessage)

	for {
		message, isOK := <-client.MessagesIn
		if !isOK {
			break
		}

		// Run our internal command handlers first
		command, commandExists := ServerCommands[message.Command]
		if commandExists {
			command.Run(client, &message)
		}

		// Dispatch any command handler
		handlers, _ = client.CommandListeners[strings.ToUpper(message.Command)]
		if handlers != nil {
			for _, handler := range handlers {
				handler(&message)
			}
		}

		// Dispatch any ALL handlers
		handlers, _ = client.CommandListeners["ALL"]
		if handlers != nil {
			for _, handler := range handlers {
				handler(&message)
			}
		}
	}

	handlers, _ = client.CommandListeners["CLOSED"]
	if handlers != nil {
		for _, handler := range handlers {
			handler(nil)
		}
	}
}

func (client *Client) JoinChannel(channel string, key string) {
	if client.Connected {
		client.WriteLine("JOIN %s %s", channel, key)
	}
}
