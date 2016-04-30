// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"errors"
	"strings"
)

var (
	errNameBadChar = errors.New("Name contained a disallowed character.")
	errNameDigit   = errors.New("The first character of a name cannot be a digit.")
	errNameSpace   = errors.New("Names cannot contain whitespace.")
	errNameNil     = errors.New("Names need to be at least one character long.")
)

func IrcName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if len(name) < 1 {
		return "", errNameNil
	}

	for _, char := range name {
		// exclude space characters
		if strings.TrimSpace(string(char)) != string(char) {
			return "", errNameSpace
		}
		// exclude other characters that mess with the protocol
		if strings.Contains(",.!@#", string(char)) {
			return "", errNameBadChar
		}
	}

	if strings.Contains("0123456789", string(name[0])) {
		return "", errNameDigit
	}

	return name, nil
}
