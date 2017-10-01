// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package ircbnc

import (
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
)

type MessageDatastore interface {
	Store(hookEvent *HookIrcRaw)
	GetFromTime(userID string, networkID string, bufferName string, timeFrom time.Time, num int) []*ircmsg.IrcMessage
	GetBeforeTime(userID string, networkID string, bufferName string, timeFrom time.Time, num int) []*ircmsg.IrcMessage
	Search(userID string, networkID string, bufferName string, timeFrom time.Time, timeTo time.Time, num int) []*ircmsg.IrcMessage

	SupportsStore() bool
	SupportsRetrieve() bool
	SupportsSearch() bool
}
