// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

// ServerConnection represents a connection to an IRC server.
type ServerConnection struct {
	Name string

	Address string
	Port    int
	UseTLS  bool
}
