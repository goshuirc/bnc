// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"github.com/goshuirc/irc-go/client"
	"github.com/tidwall/buntdb"
)

// User represents an ircbnc user.
type User struct {
	Manager *Manager
	Config  *Config
	DB      *buntdb.DB

	ID   string
	Name string

	HashedPassword []byte
	Salt           []byte
	Permissions    []string

	DefaultNick   string
	DefaultFbNick string
	DefaultUser   string
	DefaultReal   string

	Networks map[string]*ServerConnection
}

func NewUser(manager *Manager) *User {
	return &User{
		Manager:  manager,
		Networks: make(map[string]*ServerConnection),
	}
}

// StartServerConnections starts running the server connections of this user.
func (user *User) StartServerConnections(r gircclient.Reactor) {
	for _, sc := range user.Networks {
		sc.Start(r)
	}
}
