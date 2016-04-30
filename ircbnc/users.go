// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"strings"

	"github.com/DanielOaks/go-idn/idna2003/stringprep"
)

// Username takes the given name and returns a gircbnc username
func Username(name string) (string, error) {
	name, err := stringprep.Nameprep(strings.TrimSpace(name))

	if len(name) < 1 {
		return "", errNameNil
	}

	for _, char := range name {
		// exclude space characters
		if strings.TrimSpace(string(char)) != string(char) {
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
