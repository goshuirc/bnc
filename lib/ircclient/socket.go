package ircclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

type Socket struct {
	Host       string
	Port       int
	TLS        bool
	TLSConfig  *tls.Config
	Conn       net.Conn
	Connected  bool
	MessagesIn chan ircmsg.IrcMessage
}

func NewSocket() *Socket {
	return &Socket{
		MessagesIn: make(chan ircmsg.IrcMessage),
	}
}

func (socket *Socket) Connect() error {
	socket.Connected = false

	// TODO: Timeouts
	conn, err := net.Dial("tcp", net.JoinHostPort(socket.Host, strconv.Itoa(socket.Port)))
	if err != nil {
		return err
	}

	socket.Connected = true
	socket.Conn = conn
	go socket.readInput()

	return nil
}

func (socket *Socket) Close() error {
	if !socket.Connected {
		return socket.Conn.Close()
	}

	return nil
}

func (socket *Socket) readInput() {
	reader := bufio.NewReader(socket.Conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.Trim(line, "\r\n")
		message, parseErr := ircmsg.ParseLine(line)
		if parseErr == nil {
			socket.MessagesIn <- message
		}
	}

	socket.Connected = false
	close(socket.MessagesIn)
}

// WriteLine writes a raw IRC line to the server. Auto appends \n
func (socket *Socket) WriteLine(format string, args ...interface{}) (int, error) {
	if !socket.Connected {
		return 0, fmt.Errorf("not connected")
	}

	line := ""

	if strings.HasSuffix(format, "\n") {
		line = fmt.Sprintf(format, args...)
	} else {
		line = fmt.Sprintf(format+"\n", args...)
	}

	return socket.Write([]byte(line))
}

func (socket *Socket) Write(p []byte) (n int, err error) {
	return socket.Conn.Write(p)
}
