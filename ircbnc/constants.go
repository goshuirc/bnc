// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionSplit is a machine-readable version identifier
var VersionSplit = []int{0, 0, 1}

// VersionType is a string representing the type of version this is
var VersionType = "alpha"

// Version is a user-readable version string
func Version() (ver string) {
	var versionSplitString []string
	for _, num := range VersionSplit {
		versionSplitString = append(versionSplitString, strconv.Itoa(num))
	}
	return fmt.Sprintf("gIRCbnc %s %s", strings.Join(versionSplitString, "."), VersionType)
}
