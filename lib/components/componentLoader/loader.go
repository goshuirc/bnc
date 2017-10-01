// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package bncComponentLoader

import (
	"github.com/goshuirc/bnc/lib"

	// Different parts of the project acting independantly
	"github.com/goshuirc/bnc/lib/components/bouncer"
	"github.com/goshuirc/bnc/lib/components/control"
	"github.com/goshuirc/bnc/lib/components/messageLogger"
)

func Run(manager *ircbnc.Manager) {
	bncComponentControl.Run(manager)
	bncComponentLogger.Run(manager)
	bncComponentBouncer.Run(manager)
}
