// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

// +build !sqlite

package bncComponentLogger

import (
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

type SqliteMessageDatastore struct {
}

func (ds *SqliteMessageDatastore) SupportsStore() bool {
	return false
}
func (ds *SqliteMessageDatastore) SupportsRetrieve() bool {
	return false
}
func (ds *SqliteMessageDatastore) SupportsSearch() bool {
	return false
}
func NewSqliteMessageDatastore(config map[string]string) *SqliteMessageDatastore {
	ds := &SqliteMessageDatastore{}
	return ds
}

func (ds *SqliteMessageDatastore) Store(event *ircbnc.HookIrcRaw) {
}

func (ds *SqliteMessageDatastore) GetFromTime(string, string, string, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
func (ds *SqliteMessageDatastore) GetBeforeTime(string, string, string, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
func (ds *SqliteMessageDatastore) Search(string, string, string, time.Time, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
