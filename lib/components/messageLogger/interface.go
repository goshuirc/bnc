package bncComponentLogger

import (
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

type MessageDatastore interface {
	Store(*ircbnc.HookIrcRaw)
	GetFromTime(string, time.Time, int) []ircmsg.IrcMessage
	GetBeforeTime(string, time.Time, int) []ircmsg.IrcMessage
	Search(string, string, time.Time, time.Time, int) []ircmsg.IrcMessage

	//SupportsStore bool
	//SupportsRetrieve bool
	//SupportsSearch bool
}
