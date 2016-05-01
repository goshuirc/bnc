// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
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
func (b *Bouncer) Run() {
	fmt.Println("Running bouncer!")
}
