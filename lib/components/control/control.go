package bncComponentControl

import (
	"strings"

	"log"

	"github.com/goshuirc/bnc/lib"
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
	log.Println("Inside controls component", msg.Command, msg.Params[0])
	if msg.Command != "PRIVMSG" || msg.Params[0] != CONTROL_NICK {
		return
	}

	// Stop the message from being sent upstream
	event.Halt = true

	if strings.HasPrefix("listnetworks", msg.Params[1]) {
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
}
