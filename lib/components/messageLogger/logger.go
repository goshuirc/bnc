package bncComponentLogger

import (
	"log"
	"time"

	"github.com/goshuirc/bnc/lib"
)

var stores []MessageDatastore

func Run(manager *ircbnc.Manager) {
	for logType, logConf := range manager.Config.Bouncer.Logging {
		if logType == "file" {
			log.Println("Starting message logger: " + logType)
			store := NewFileMessageDatastore(logConf)
			stores = append(stores, &store)
		} else if logType == "sqlite" {
			log.Println("Starting message logger: " + logType)
			store := NewSqliteMessageDatastore(logConf)
			stores = append(stores, &store)
		}
	}

	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
	manager.Bus.Register(ircbnc.HookStateSentName, onStateSent)
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

func onStateSent(hook interface{}) {
	event := hook.(*ircbnc.HookStateSent)

	var store MessageDatastore
	for _, currentStore := range stores {
		if currentStore.SupportsRetrieve() {
			store = currentStore
			break
		}
	}

	if store == nil {
		return
	}

	// Only send buffer history if we're connected to a network
	if event.Server == nil {
		return
	}

	for _, channel := range event.Server.Channels {
		msgs := store.GetBeforeTime(event.Listener.User.ID, event.Server.Name, channel.Name, time.Now(), 50)
		for _, message := range msgs {
			line, err := message.Line()
			if err != nil {
				log.Println("Error building message from storage:", err.Error())
				continue
			}
			event.Listener.SendLine(line)
		}
	}
}
