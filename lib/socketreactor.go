// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"fmt"
	"net"
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

// SocketReactor listens to a socket using the IRC protocol, processes events,
// and also sends IRC lines out of that socket.
type SocketReactor struct {
	receiveLines  chan string
	ReceiveEvents chan Message
	SendLines     chan string
	socket        Socket

	processIncomingLine func(line string) bool
}

// NewSocketReactor returns a new SocketReactor.
func NewSocketReactor(conn net.Conn, pilHandle func(line string) bool) SocketReactor {
	return SocketReactor{
		receiveLines:  make(chan string),
		ReceiveEvents: make(chan Message),
		SendLines:     make(chan string),
		socket:        NewSocket(conn),

		processIncomingLine: pilHandle,
	}
}

// Start creates and starts running the necessary event loops.
func (reactor *SocketReactor) Start() {
	go reactor.RunEvents()
	go reactor.RunSocketSender()
	go reactor.RunSocketListener()
}

// RunEvents handles received IRC lines and processes incoming commands.
func (reactor *SocketReactor) RunEvents() {
	var exiting bool
	var line string
	for {
		select {
		case line = <-reactor.receiveLines:
			if line != "" {
				fmt.Println("<- ", strings.TrimRight(line, "\r\n"))
				exiting = reactor.processIncomingLine(line)
				if exiting {
					reactor.socket.Close()
					break
				}
			}
		}
	}
	// empty the receiveLines queue
	select {
	case <-reactor.receiveLines:
		// empty
	default:
		// empty
	}
}

// RunSocketSender sends lines to the IRC socket.
func (reactor *SocketReactor) RunSocketSender() {
	var err error
	var line string
	for {
		line = <-reactor.SendLines
		err = reactor.socket.Write(line)
		fmt.Println(" ->", strings.TrimRight(line, "\r\n"))
		if err != nil {
			break
		}
	}
}

// RunSocketListener receives lines from the IRC socket.
func (reactor *SocketReactor) RunSocketListener() {
	var errConn error
	var line string

	for {
		line, errConn = reactor.socket.Read()
		reactor.receiveLines <- line
		if errConn != nil {
			break
		}
	}
	if !reactor.socket.Closed {
		reactor.Send(nil, "", "ERROR", "Closing connection")
		reactor.socket.Close()
	}
}

// Send sends an IRC line to the listener.
func (reactor *SocketReactor) Send(tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) error {
	ircmsg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := ircmsg.Line()
	if err != nil {
		return err
	}
	reactor.SendLines <- line
	return nil
}
