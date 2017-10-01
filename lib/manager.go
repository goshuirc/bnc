// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2014-2015 Edmund Huber
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
)

var (
	// QuitSignals is the list of signals we quit on
	//TODO(dan): Rehash on one of these signals instead, same as Oragono.
	QuitSignals = []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT}
	// BNC: The global instance of Manager.
	// TODO: NewManager() sets this each time it's run. It's only run once so no issue.. but it's not tidy
	BNC *Manager
)

// Manager handles the different components that keep GoshuBNC spinning.
type Manager struct {
	Config   *Config
	Ds       DataStoreInterface
	Messages MessageDatastore

	Users     map[string]*User
	Listeners []net.Listener

	newConns    chan net.Conn
	quitSignals chan os.Signal

	Source       string
	StatusNick   string
	StatusSource string

	Bus HookEmitter

	Salt []byte
}

// NewManager create a new IRC bouncer from the given config and database.
func NewManager(config *Config, ds DataStoreInterface) *Manager {
	m := &Manager{}
	BNC = m

	m.Bus = MakeHookEmitter()
	m.Config = config

	m.Ds = ds

	m.newConns = make(chan net.Conn)
	m.quitSignals = make(chan os.Signal, len(QuitSignals))

	m.Users = make(map[string]*User)

	// source on our outgoing message/status bot/etc
	m.StatusNick = "*status"
	m.Source = "irc.goshubnc"
	m.StatusSource = fmt.Sprintf("%s!bnc@%s", m.StatusNick, m.Source)

	return m
}

// Run starts the bouncer, creating the listeners and server connections.
func (m *Manager) Run() error {

	// load users
	users := m.Ds.GetAllUsers()
	for _, user := range users {
		m.Users[user.ID] = user
		m.Users[user.ID].StartServerConnections()
	}

	// open listeners
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
				log.Fatal(fmt.Sprintf("TLS listen error on `%s`: %s", address, err.Error()))
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
			log.Fatal("Shutting down! (TODO: write real shutdown code)")
			done = true
		case conn := <-m.newConns:
			go NewListener(m, conn)
		}
	}

	return nil
}
