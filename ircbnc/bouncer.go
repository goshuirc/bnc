// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license
// some code in here is taken from Ergonomadic/Oragono

package ircbnc

import (
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/DanielOaks/girc-go/client"
)

var (
	// ServerSignals is the list of signals we break on
	ServerSignals = []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT}
)

// Bouncer represents an IRC bouncer.
type Bouncer struct {
	Config *Config
	DB     *sql.DB

	Users     map[string]User
	Listeners []net.Listener

	newConns chan net.Conn
	signals  chan os.Signal

	Salt []byte
}

// NewBouncer create a new IRC bouncer from the given config and ircbnc database.
func NewBouncer(config *Config, db *sql.DB) (*Bouncer, error) {
	var b Bouncer
	b.Config = config
	b.DB = db

	b.newConns = make(chan net.Conn)
	b.signals = make(chan os.Signal, len(ServerSignals))

	b.Users = make(map[string]User)

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

	// open listeners and wait
	for _, address := range b.Config.Bouncer.Listeners {
		config, listenTLS := b.Config.Bouncer.TLSListeners[address]

		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal(address, "listen error: ", err)
		}

		tlsString := "plaintext"
		if listenTLS {
			tlsConfig, err := config.Config()
			if err != nil {
				log.Fatal(address, "tls listen error: ", err)
			}
			listener = tls.NewListener(listener, tlsConfig)
			tlsString = "TLS"
		}
		fmt.Println(fmt.Sprintf("listening on %s using %s.", address, tlsString))

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Printf("%s accept error: %s", address, err)
				}
				fmt.Printf("%s accept: %s", address, conn.RemoteAddr())

				b.newConns <- conn
			}
		}()

		b.Listeners = append(b.Listeners, listener)
	}

	// and wait
	var done bool
	for !done {
		select {
		case <-b.signals:
			//TODO(dan): Write real shutdown code
			log.Fatal("Shutting down! (TODO: write real code)")
			done = true
		case conn := <-b.newConns:
			NewListener(b, conn)
		}
	}

	return nil
}
