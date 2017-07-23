// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"fmt"
)

const (
	// SemVer is the semantic version of GoshuBNC.
	SemVer = "0.1.0-unreleased"
)

var (
	// Ver is the full version of GoshuBNC, used in responses to clients.
	Ver = fmt.Sprintf("goshubnc-%s", SemVer)
)
