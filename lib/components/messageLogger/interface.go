package bncComponentLogger

import (
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

type MessageDatastore interface {
	Store(hookEvent *ircbnc.HookIrcRaw)
	GetFromTime(userID string, networkID string, bufferName string, timeFrom time.Time, num int) []*ircmsg.IrcMessage
	GetBeforeTime(userID string, networkID string, bufferName string, timeFrom time.Time, num int) []*ircmsg.IrcMessage
	Search(userID string, networkID string, bufferName string, timeFrom time.Time, timeTo time.Time, num int) []*ircmsg.IrcMessage

	SupportsStore() bool
	SupportsRetrieve() bool
	SupportsSearch() bool
}
