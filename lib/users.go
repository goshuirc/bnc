// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

// User represents an ircbnc user.
type User struct {
	Manager *Manager
	Config  *Config

	ID   string
	Name string
	Role string

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
func (user *User) StartServerConnections() {
	for _, sc := range user.Networks {
		if sc.Enabled {
			go sc.Connect()
		}
	}
}
