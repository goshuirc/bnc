// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2014-2015 Edmund Huber
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/goshuirc/irc-go/client"
	"github.com/tidwall/buntdb"
)

var (
	// QuitSignals is the list of signals we quit on
	//TODO(dan): Rehash on one of these signals instead, same as Oragono.
	QuitSignals = []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT}
)

// Manager handles the different components that keep GoshuBNC spinning.
type Manager struct {
	Config *Config
	DB     *buntdb.DB

	Users     map[string]*User
	Listeners []net.Listener

	newConns    chan net.Conn
	quitSignals chan os.Signal

	Source       string
	StatusSource string

	Salt []byte
}

// NewManager create a new IRC bouncer from the given config and database.
func NewManager(config *Config, db *buntdb.DB) (*Manager, error) {
	var m Manager
	m.Config = config
	m.DB = db

	m.newConns = make(chan net.Conn)
	m.quitSignals = make(chan os.Signal, len(QuitSignals))

	m.Users = make(map[string]*User)

	// source on our outgoing message/status bot/etc
	m.Source = "irc.goshubnc"
	m.StatusSource = fmt.Sprintf("*status!bnc@%s", m.Source)

	err := db.View(func(tx *buntdb.Tx) error {
		saltString, err := tx.Get(KeySalt)
		if err != nil {
			return fmt.Errorf("Could not get salt string: %s", err.Error())
		}

		m.Salt, err = base64.StdEncoding.DecodeString(saltString)
		if err != nil {
			return fmt.Errorf("Could not decode b64'd salt: %s", err.Error())
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Creating new bouncer failed: %s", err.Error())
	}

	return &m, nil
}

// Run starts the bouncer, creating the listeners and server connections.
func (m *Manager) Run() error {
	// create reactor
	scReactor := gircclient.NewReactor()

	// load users
	err := m.DB.Update(func(tx *buntdb.Tx) error {
		var userIDs []string

		tx.DescendKeys("user.info *", func(key, value string) bool {
			userIDs = append(userIDs, strings.TrimPrefix(key, "user.info "))
			return true // continue looping through keys
		})

		// add users to bouncer
		for _, id := range userIDs {
			user, err := loadUser(m.Config, m.DB, tx, id)
			if err != nil {
				return fmt.Errorf("Could not run bouncer (loading user from db): %s", err.Error())
			}

			m.Users[id] = user
		}

		// start server connections for all users
		for _, id := range userIDs {
			m.Users[id].StartServerConnections(scReactor)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// open listeners and wait
	for _, address := range m.Config.Bouncer.Listeners {
		config, listenTLS := m.Config.Bouncer.TLSListeners[address]

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
					fmt.Println(fmt.Sprintf("%s accept error: %s", address, err))
				}
				fmt.Println(fmt.Sprintf("%s accept: %s", address, conn.RemoteAddr()))

				m.newConns <- conn
			}
		}()

		m.Listeners = append(m.Listeners, listener)
	}

	// and wait
	var done bool
	for !done {
		select {
		case <-m.quitSignals:
			//TODO(dan): Write real shutdown code
			log.Fatal("Shutting down! (TODO: write real code)")
			done = true
		case conn := <-m.newConns:
			NewListener(m, conn)
		}
	}

	return nil
}
