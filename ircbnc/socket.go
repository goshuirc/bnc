// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type Socket struct {
	closed bool
	conn   net.Conn
	reader *bufio.Reader
	buffer string
}

func NewSocket(conn net.Conn) Socket {
	return Socket{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (socket *Socket) Close() {
	if socket.closed {
		return
	}
	socket.closed = true
	socket.conn.Close()
}

func (socket *Socket) Read() (string, error) {
	if socket.closed {
		return "", io.EOF
	}

	lineBytes, err := socket.reader.ReadBytes('\n')

	// convert bytes to string
	line := string(lineBytes[:])

	// read last message properly (such as ERROR/QUIT/etc), just fail next reads/writes
	if err == io.EOF {
		socket.Close()
	} else if err != nil {
		return "", err
	}

	return line, nil
}

func (socket *Socket) Write(line string) error {
	if socket.closed {
		return io.EOF
	}

	// write data
	_, err := fmt.Fprintf(socket.conn, line)
	if err != nil {
		socket.Close()
		return err
	}
	return nil
}
