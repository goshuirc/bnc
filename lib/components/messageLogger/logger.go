package bncComponentLogger

import (
	"log"

	"github.com/goshuirc/bnc/lib"
)

var stores []MessageDatastore

func Run(manager *ircbnc.Manager) {
	for logType, logConf := range manager.Config.Bouncer.Logging {
		if logType == "file" {
			log.Println("Starting message logger: " + logType)
			store := NewFileMessageDatastore(logConf)
			stores = append(stores, store)
		}
	}

	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
}

func onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)

	// Only deal with messages from logged in users
	if event.User == nil {
		return
	}

	for _, store := range stores {
		store.Store(event)
	}
}
