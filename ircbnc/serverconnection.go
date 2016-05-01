// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/DanielOaks/girc-go/client"
	"github.com/DanielOaks/girc-go/eventmgr"
	"github.com/DanielOaks/girc-go/ircfmt"
)

// ServerConnectionAddress represents an address a ServerConnection can join.
type ServerConnectionAddress struct {
	Address string
	Port    int
	UseTLS  bool
}

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name      string
	User      User
	Connected bool

	Nickname   string
	FbNickname string
	Username   string
	Realname   string
	Channels   map[string]string

	Password  string
	Addresses []ServerConnectionAddress
}

// LoadServerConnection loads the given server connection from our database.
func LoadServerConnection(name string, user User, db *sql.DB) (*ServerConnection, error) {
	var sc ServerConnection
	sc.Name = name
	sc.User = user

	row := db.QueryRow(`SELECT nickname, fallback_nickname, username, realname, password FROM server_connections WHERE user_id = ? AND name = ?`,
		user.ID, name)
	err := row.Scan(&sc.Nickname, &sc.FbNickname, &sc.Username, &sc.Realname, &sc.Password)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (loading sc details from db): %s", err.Error())
	}

	// set default values
	if sc.Nickname == "" {
		sc.Nickname = user.DefaultNick
	}
	if sc.FbNickname == "" {
		sc.FbNickname = user.DefaultFbNick
	}
	if sc.Username == "" {
		sc.Username = user.DefaultUser
	}
	if sc.Realname == "" {
		sc.Realname = user.DefaultReal
	}

	// load channels
	sc.Channels = make(map[string]string)
	rows, err := db.Query(`SELECT name, key FROM server_connection_channels WHERE user_id = ? AND sc_name = ?`,
		user.ID, name)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (loading address details from db): %s", err.Error())
	}
	for rows.Next() {
		var name, key string
		rows.Scan(&name, &key)

		sc.Channels[name] = key
	}

	// load addresses
	rows, err = db.Query(`SELECT address, port, use_tls FROM server_connection_addresses WHERE user_id = ? AND sc_name = ?`,
		user.ID, name)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (loading address details from db): %s", err.Error())
	}
	for rows.Next() {
		var address, portString string
		var useTLS bool

		rows.Scan(&address, &portString, &useTLS)

		port, err := strconv.Atoi(portString)
		if err != nil {
			return nil, fmt.Errorf("Could not create new ServerConnection (port did not load correctly): %s", err.Error())
		} else if port < 1 || port > 65535 {
			return nil, fmt.Errorf("Could not create new ServerConnection (port %d is not valid)", port)
		}

		var newAddress ServerConnectionAddress
		newAddress.Address = address
		newAddress.Port = port
		newAddress.UseTLS = useTLS
		sc.Addresses = append(sc.Addresses, newAddress)
	}

	return &sc, nil
}

// rawHandler prints raw messages to and from the server.
//TODO(dan): This is only VERY INITIAL, for use while we are debugging.
func rawHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*gircclient.ServerConnection)
	direction := info["direction"].(string)
	line := info["data"].(string)

	var arrow string
	if direction == "in" {
		arrow = "<- "
	} else {
		arrow = " ->"
	}

	fmt.Println(sc.Name, arrow, ircfmt.Escape(strings.Trim(line, "\r\n")))
}

// Start opens and starts connecting to the server.
func (sc *ServerConnection) Start(reactor gircclient.Reactor) {
	name := fmt.Sprintf("%s %s", sc.User.ID, sc.Name)
	server := reactor.CreateServer(name)

	server.InitialNick = sc.Nickname
	server.InitialUser = sc.Username
	server.InitialRealName = sc.Realname
	server.ConnectionPass = sc.Password
	//TODO(dan): Fallback Nick

	server.RegisterEvent("in", "raw", rawHandler, 0)
	server.RegisterEvent("out", "raw", rawHandler, 0)

	var err error
	for _, address := range sc.Addresses {
		fullAddress := net.JoinHostPort(address.Address, strconv.Itoa(address.Port))

		err = server.Connect(fullAddress, address.UseTLS, nil)
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Println("ERROR: Could not connect to", name, err.Error())
		return
	}

	go server.ReceiveLoop()
}
