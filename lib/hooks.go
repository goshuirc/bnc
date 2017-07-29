package ircbnc

import (
	"github.com/goshuirc/irc-go/ircmsg"
)

var HookIrcClientRawName = "irc.client.raw"

type HookIrcClientRaw struct {
	Listener *Listener
	Raw      string
	Message  ircmsg.IrcMessage
	Halt     bool
}
