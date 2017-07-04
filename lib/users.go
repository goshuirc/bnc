// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/goshuirc/irc-go/client"
)

// User represents an ircbnc user.
type User struct {
	Config *Config
	DB     *sql.DB

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
func loadUser(config *Config, db *sql.DB, id string) (*User, error) {
	var user User
	user.ID = id
	user.Name = id //TODO(dan): Store Name and ID separately in the future if we want to
	user.Config = config
	user.DB = db

	user.Networks = make(map[string]*ServerConnection)

	userRow := db.QueryRow(`SELECT password, salt, default_nickname, default_fallback_nickname, default_username, default_realname FROM users WHERE id = ?`,
		id)
	var saltString string
	err := userRow.Scan(&user.HashedPassword, &saltString, &user.DefaultNick, &user.DefaultFbNick, &user.DefaultUser, &user.DefaultReal)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (scanning user info from db): %s", err.Error())
	}

	user.Salt, err = base64.StdEncoding.DecodeString(saltString)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (decoding salt): %s", err.Error())
	}

	rows, err := db.Query(`SELECT name FROM server_connections WHERE user_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (loading sc names from db): %s", err.Error())
	}
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, fmt.Errorf("Could not load user (scanning sc names from db): %s", err.Error())
		}

		sc, err := LoadServerConnection(name, user, user.DB)
		if err != nil {
			return nil, fmt.Errorf("Could not load user (loading sc): %s", err.Error())
		}

		user.Networks[name] = sc
	}

	return &user, nil
}

// StartServerConnections starts running the server connections of this user.
func (user *User) StartServerConnections(r gircclient.Reactor) {
	for _, sc := range user.Networks {
		sc.Start(r)
	}
}
