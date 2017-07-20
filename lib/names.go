// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/text/secure/precis"
)

var (
	errNameBadChar = errors.New("Name contained a disallowed character")
	errNameDigit   = errors.New("The first character of a name cannot be a digit")
	errNameSpace   = errors.New("Names cannot contain whitespace")
	errNameNil     = errors.New("Names need to be at least one character long")
)

// IrcName returns a name appropriate for IRC use (nick/user/channel), or an error if the name is bad.
func IrcName(name string, isChannel bool) (string, error) {
	name = strings.TrimSpace(name)

	if len(name) < 1 {
		return "", errNameNil
	}

	for _, char := range name {
		// exclude space characters
		if unicode.IsSpace(char) {
			return "", errNameSpace
		}
		// exclude other characters that mess with the protocol
		if isChannel {
			if strings.Contains(",?*", string(char)) {
				return "", errNameBadChar
			}
		} else {
			if strings.Contains(",.!@#?*", string(char)) {
				return "", errNameBadChar
			}
		}
	}

	return name, nil
}

// BncName takes the given name and returns a casefolded name appropriate for use with ircbnc.
// This includes usernames, network names, etc.
func BncName(name string) (string, error) {
	name, err := precis.UsernameCaseMapped.CompareKey(name)

	if len(name) < 1 {
		return "", errNameNil
	}

	for _, char := range name {
		// exclude space characters
		if unicode.IsSpace(char) {
			return "", errNameSpace
		}
		// exclude other characters that seem like they could be bad
		if strings.Contains(",.=!@#*%&$/\\", string(char)) {
			return "", errNameBadChar
		}
	}

	if strings.Contains("0123456789", string(name[0])) {
		return "", errNameDigit
	}

	return name, err
}
