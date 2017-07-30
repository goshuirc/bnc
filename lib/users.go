// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"encoding/base64"
	"fmt"

	"encoding/json"

	"strings"

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

// LoadUser returns the given user.
func loadUser(manager *Manager, tx *buntdb.Tx, id string) (*User, error) {
	var user User
	user.ID = id
	user.Name = id //TODO(dan): Store Name and ID separately in the future if we want to
	user.Manager = manager
	user.Config = manager.Config
	user.DB = manager.DB

	user.Networks = make(map[string]*ServerConnection)

	// load user info
	infoString, err := tx.Get(fmt.Sprintf(KeyUserInfo, id))
	if err != nil {
		return nil, fmt.Errorf("Could not load user (loading user info from db): %s", err.Error())
	}
	ui := &UserInfo{}
	err = json.Unmarshal([]byte(infoString), ui)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (unmarshalling user info from db): %s", err.Error())
	}

	user.Salt, err = base64.StdEncoding.DecodeString(ui.EncodedSalt)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (decoding salt): %s", err.Error())
	}

	//TODO(dan): Make the below both have the same named fields
	user.HashedPassword, err = base64.StdEncoding.DecodeString(ui.EncodedPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (decoding password): %s", err.Error())
	}
	user.DefaultNick = ui.DefaultNick
	user.DefaultFbNick = ui.DefaultNickFallback
	user.DefaultUser = ui.DefaultUsername
	user.DefaultReal = ui.DefaultRealname

	// load server connections
	var scError error
	tx.DescendKeys(fmt.Sprintf("user.server.info %s *", user.ID), func(key, value string) bool {
		name := strings.TrimPrefix(key, fmt.Sprintf("user.server.info %s ", user.ID))

		sc, err := LoadServerConnection(name, user, tx)
		if err != nil {
			scError = fmt.Errorf("Could not load user (loading sc): %s", err.Error())
			return false
		}

		user.Networks[name] = sc

		return true
	})
	if scError != nil {
		return nil, scError
	}

	return &user, nil
}

// StartServerConnections starts running the server connections of this user.
func (user *User) StartServerConnections(r gircclient.Reactor) {
	for _, sc := range user.Networks {
		sc.Start(r)
	}
}
