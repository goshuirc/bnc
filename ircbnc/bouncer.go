// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/DanielOaks/girc-go/client"
)

// Bouncer represents an IRC bouncer.
type Bouncer struct {
	Config *Config
	DB     *sql.DB

	Users     map[string]User
	Listeners map[int]net.Listener

	Salt []byte
}

// NewBouncer create a new IRC bouncer from the given config and ircbnc database.
func NewBouncer(config *Config, db *sql.DB) (*Bouncer, error) {
	var b Bouncer
	b.Config = config
	b.DB = db

	b.Users = make(map[string]User)
	b.Listeners = make(map[int]net.Listener)

	saltRow := db.QueryRow(`SELECT value FROM ircbnc WHERE key = ?`, "salt")
	var saltString string
	err := saltRow.Scan(&saltString)
	if err != nil {
		return nil, fmt.Errorf("Creating new bouncer failed (could not scan out salt string):", err.Error())
	}

	b.Salt, err = base64.StdEncoding.DecodeString(saltString)
	if err != nil {
		return nil, fmt.Errorf("Creating new bouncer failed (could not decode b64'd salt):", err.Error())
	}

	return &b, nil
}

// Run starts the bouncer, creating the listeners and server connections.
func (b *Bouncer) Run() error {
	// create reactor
	scReactor := gircclient.NewReactor()

	// load users
	rows, err := b.DB.Query(`SELECT id FROM users`)
	if err != nil {
		return fmt.Errorf("Could not run bouncer (loading users from db): %s", err.Error())
	}
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return fmt.Errorf("Could not run bouncer (scanning user names from db): %s", err.Error())
		}

		user, err := LoadUser(b.Config, b.DB, id)
		if err != nil {
			return fmt.Errorf("Could not run bouncer (loading user from db): %s", err.Error())
		}

		user.StartServerConnections(scReactor)
	}

	return nil
}
