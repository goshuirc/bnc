package bncComponentControl

import (
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

// Nick of the controller
const CONTROL_NICK = "*goshu"
const CONTROL_PREFIX = CONTROL_NICK + "!bnc@irc.goshu"

func Run(manager *ircbnc.Manager) {
	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
}

func onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)
	if !event.FromClient {
		return
	}

	msg := event.Message
	listener := event.Listener

	if msg.Command != "PRIVMSG" || msg.Params[0] != CONTROL_NICK {
		return
	}

	// Stop the message from being sent upstream
	event.Halt = true

	parts := strings.Split(msg.Params[1], " ")
	command := strings.ToLower(parts[0])
	params := parts[1:]

	switch command {
	case "listnetworks":
		commandListNetworks(listener, params, msg)
	}
}

func commandListNetworks(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	table := NewTable()
	table.SetHeader([]string{"Name", "Nick", "Connected"})

	for _, network := range listener.User.Networks {
		connected := "No"
		if network.Connected {
			connected = "Yes"
		}
		table.Append([]string{network.Name, network.Nickname, connected})
	}

	table.RenderToListener(listener, CONTROL_PREFIX, "PRIVMSG")
}
