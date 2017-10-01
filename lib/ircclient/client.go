// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package ircclient

import (
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircmsg"
)

/**
 * ClientCaps holds the capabilities between the client and server
 */
type ClientCaps struct {
	Wanted    []string
	Enabled   map[string]string
	Available map[string]string
}

// CommonCaps returns a slice of caps that both the client and server support
func (caps *ClientCaps) CommonCaps() []string {
	var common []string

	for _, wantedCap := range caps.Wanted {
		_, exists := caps.Available[wantedCap]
		if exists {
			common = append(common, wantedCap)
		}
	}

	return common
}

// IsEnabled checks if a particular cap is enabled for this connection
func (caps *ClientCaps) IsEnabled(cap string) bool {
	_, exists := caps.Enabled[cap]
	return exists
}

/**
 * Client is the IRC client
 */
type Client struct {
	sync.RWMutex
	Socket
	Nick             string
	Username         string
	Realname         string
	Password         string
	BindHost         string
	Caps             *ClientCaps
	Supported        map[string]string
	HasRegistered    bool
	CommandListeners map[string][]func(*ircmsg.IrcMessage)
}

func NewClient() *Client {
	client := &Client{
		Socket:           *NewSocket(),
		Supported:        make(map[string]string),
		CommandListeners: make(map[string][]func(*ircmsg.IrcMessage)),
	}

	client.Caps = &ClientCaps{
		Enabled:   make(map[string]string),
		Available: make(map[string]string),
	}

	client.Caps.Wanted = append(
		client.Caps.Wanted,
		"account-notify",
		"away-notify",
		"extended-join",
		// "multi-prefix",
		// "sasl",
		"account-tag",
		// "cap-notify",
		// "chghost",
		"invite-notify",
		"server-time",
		"userhost-in-names",
	)

	return client
}

func (client *Client) Connect() error {
	err := client.Socket.Connect()
	if err != nil {
		return err
	}

	go client.messageDispatcher()

	if client.Password != "" {
		client.WriteLine("PASS " + client.Password)
	}
	client.WriteLine("CAP LS 302")
	client.WriteLine("NICK %s", client.Nick)
	client.WriteLine("USER %s 0 * :%s", client.Username, client.Realname)

	return nil
}

func (client *Client) HandleCommand(command string, fn func(*ircmsg.IrcMessage)) {
	client.Lock()
	defer client.Unlock()

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
			shouldHalt := command.Run(client, &message)
			if shouldHalt {
				continue
			}
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

	client.Lock()
	client.HasRegistered = false
	client.Unlock()
}

func (client *Client) JoinChannel(channel string, key string) {
	if client.Connected {
		client.WriteLine("JOIN %s %s", channel, key)
	}
}
