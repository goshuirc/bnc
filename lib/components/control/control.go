package bncComponentControl

import (
	"strings"

	"log"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/eventmgr"
	"github.com/goshuirc/irc-go/ircmsg"
)

// Nick of the controller
const CONTROL_NICK = "*goshu"
const CONTROL_PREFIX = CONTROL_NICK + "!bnc@irc.goshu"

func Run(manager *ircbnc.Manager) {
	manager.Bus.Attach("irc.client.raw", onMessage, 0)
}

func onMessage(event string, info eventmgr.InfoMap) {
	listener := info["listener"].(*ircbnc.Listener)
	msg := info["message"].(ircmsg.IrcMessage)
	log.Println("Inside controls component", msg.Command, msg.Params[0])
	if msg.Command != "PRIVMSG" || msg.Params[0] != CONTROL_NICK {
		return
	}

	// Stop the message from being sent upstream
	info["halt"] = true

	if strings.HasPrefix("listnetworks", msg.Params[1]) {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "Listing networks")
	}
}
