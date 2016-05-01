// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"database/sql"
	"fmt"
	"strconv"
)

// ServerConnectionAddress represents an address a ServerConnection can join.
type ServerConnectionAddress struct {
	Address string
	Port    int
	UseTLS  bool
}

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name string

	Nick   string
	FbNick string
	User   string
	Real   string

	Password  string
	Addresses []ServerConnectionAddress
}

// LoadServerConnection loads the given server connection from our database.
func LoadServerConnection(name string, user User, db *sql.DB) (*ServerConnection, error) {
	var sc ServerConnection
	sc.Name = name

	row := db.QueryRow(`SELECT nickname, fallback_nickname, username, realname, password FROM server_connections WHERE user_id = ? AND name = ?`,
		user.ID, name)
	err := row.Scan(&sc.Nick, &sc.FbNick, &sc.User, &sc.Real, &sc.Password)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (loading sc details from db): %s", err.Error())
	}

	rows, err := db.Query(`SELECT address, port, use_tls FROM server_connection_addresses WHERE user_id = ? AND sc_name = ?`,
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
	}

	return &sc, nil
}
