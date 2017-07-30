package bncComponentLoader

import (
	"github.com/goshuirc/bnc/lib"

	// Different parts of the project acting independantly
	"github.com/goshuirc/bnc/lib/components/control"
	"github.com/goshuirc/bnc/lib/components/messageLogger"
)

func Run(manager *ircbnc.Manager) {
	bncComponentControl.Run(manager)
	bncComponentLogger.Run(manager)
}
