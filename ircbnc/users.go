// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

// User represents an ircbnc user.
type User struct {
	ID          string
	Name        string
	Permissions []string

	DefaultNick   string
	DefaultFbNick string
	DefaultUser   string
	DefaultReal   string

	Networks map[string]ServerConnection
}
