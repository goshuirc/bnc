// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package bncComponentLogger

import (
	"crypto/rand"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

const MaxRetrieveSize int = 50

func Run(manager *ircbnc.Manager) {
	store, _ := getMessageDataStoreInstance(manager.Config)
	if store == nil {
		return
	}

	manager.Messages = store
	l := &Logger{
		Manager: manager,
	}
	l.RegisterHooks()
}

func getMessageDataStoreInstance(config *ircbnc.Config) (ircbnc.MessageDatastore, string) {
	loggingConfig := config.Bouncer.Logging
	storageType, _ := loggingConfig["type"]

	var store ircbnc.MessageDatastore

	if storageType == "file" {
		store = NewFileMessageDatastore(loggingConfig)
	} else if storageType == "sqlite" {
		store = NewSqliteMessageDatastore(loggingConfig)
	} else {
		// No recognised storage type. Blank it off
		storageType = ""
	}

	return store, storageType
}

type Logger struct {
	Manager *ircbnc.Manager
}

func (logger *Logger) RegisterHooks() {
	logger.Manager.Bus.Register(ircbnc.HookIrcRawName, logger.onMessage)
	logger.Manager.Bus.Register(ircbnc.HookStateSentName, logger.onStateSent)
	logger.Manager.Bus.Register(ircbnc.HookNewListenerName, logger.onNewListener)
}

func (logger *Logger) onNewListener(hook interface{}) {
	event := hook.(*ircbnc.HookNewListener)
	if logger.Manager.Messages.SupportsRetrieve() {
		event.Listener.ExtraISupports["CHATHISTORY"] = strconv.Itoa(MaxRetrieveSize)
	}
}

func (logger *Logger) onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)

	// Only deal with messages from logged in users
	if event.User == nil {
		return
	}

	logger.Manager.Messages.Store(event)

	if event.Message.Command == "CHATHISTORY" {
		event.Halt = true
		logger.handleChatHistory(event.Listener, &event.Message)
	}
}

// Send playback after a client has had its state sent
func (logger *Logger) onStateSent(hook interface{}) {
	event := hook.(*ircbnc.HookStateSent)

	// BOUNCER capable clients will use CHATHISTORY when needed
	if event.Listener.IsCapEnabled("bouncer") {
		return
	}

	store := logger.Manager.Messages
	if !store.SupportsRetrieve() {
		return
	}

	// Only send buffer history if we're connected to a network
	if event.Server == nil {
		return
	}

	for _, buffer := range event.Server.Buffers {
		msgs := store.GetBeforeTime(event.Listener.User.ID, event.Server.Name, buffer.Name, time.Now(), 50)
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

func (logger *Logger) handleChatHistory(listener *ircbnc.Listener, msg *ircmsg.IrcMessage) {
	if !listener.IsCapEnabled("batch") || listener.ServerConnection == nil {
		return
	}

	if len(msg.Params) < 3 {
		return
	}

	store := logger.Manager.Messages
	if !store.SupportsRetrieve() {
		return
	}

	target := msg.Params[0]
	start := msg.Params[1]
	end := msg.Params[2]

	startParts := strings.SplitN(start, "=", 2)
	if len(startParts) != 2 {
		return
	}
	if startParts[0] != "timestamp" {
		log.Println("Unsupported starting point for CHATHISTORY: " + startParts[0])
		return
	}

	timeFrom, timeErr := time.Parse(time.RFC3339, startParts[1])
	if timeErr != nil {
		log.Println("Error parsing date for CHATHISTORY: " + timeErr.Error())
		return
	}

	endParts := strings.SplitN(end, "=", 2)
	if len(endParts) != 2 {
		return
	}
	if endParts[0] != "message_count" {
		log.Println("Only message_count ending point is supported for CHATHISTORY")
		return
	}

	numMessages, _ := strconv.Atoi(endParts[1])
	if numMessages > MaxRetrieveSize {
		numMessages = MaxRetrieveSize
	}
	if numMessages < -MaxRetrieveSize {
		numMessages = -MaxRetrieveSize
	}

	for _, buffer := range listener.ServerConnection.Buffers {
		// If target == * then send all available buffers
		if target != "*" && strings.ToLower(target) != strings.ToLower(buffer.Name) {
			continue
		}

		var msgs []*ircmsg.IrcMessage
		if numMessages < 0 {
			msgs = store.GetBeforeTime(
				listener.User.ID,
				listener.ServerConnection.Name,
				buffer.Name,
				timeFrom,
				numMessages*-1,
			)
		} else {
			msgs = store.GetFromTime(
				listener.User.ID,
				listener.ServerConnection.Name,
				buffer.Name,
				timeFrom,
				numMessages,
			)
		}

		batchId := makeBatchId()
		listener.Send(nil, "", "BATCH", "+"+batchId, "chathistory", buffer.Name)

		for _, message := range msgs {
			message.Tags["batch"] = ircmsg.MakeTagValue(batchId)
			line, err := message.Line()
			if err != nil {
				log.Println("Error building message from storage:", err.Error())
				continue
			}
			listener.SendLine(line)
		}

		listener.Send(nil, "", "BATCH", "-"+batchId)
	}
}

func makeBatchId() string {
	length := 8
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	id := fmt.Sprintf("%X", b)
	return id
}
