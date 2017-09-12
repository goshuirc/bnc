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

var stores []MessageDatastore

const MaxRetrieveSize int = 50

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
	manager.Bus.Register(ircbnc.HookNewListenerName, onNewListener)
}

func onNewListener(hook interface{}) {
	event := hook.(*ircbnc.HookNewListener)
	event.Listener.ExtraISupports["CHATHISTORY"] = strconv.Itoa(MaxRetrieveSize)
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

	if event.Message.Command == "CHATHISTORY" {
		event.Halt = true
		handleChatHistory(event.Listener, &event.Message)
	}
}

func onStateSent(hook interface{}) {
	event := hook.(*ircbnc.HookStateSent)

	// BOUNCER capable clients will use CHATHISTORY when needed
	if event.Listener.IsCapEnabled("BOUNCER") {
		return
	}

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

func handleChatHistory(listener *ircbnc.Listener, msg *ircmsg.IrcMessage) {
	if !listener.IsCapEnabled("batch") {
		return
	}

	if len(msg.Params) < 3 {
		return
	}

	var store MessageDatastore
	for _, s := range stores {
		if s.SupportsRetrieve() {
			store = s
			break
		}
	}

	target := msg.Params[0]
	start := msg.Params[1]
	end := msg.Params[2]

	startParts := strings.SplitN(start, "=", 2)
	if len(startParts) != 2 {
		return
	}
	if startParts[0] != "timestamp" {
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
	}

	numMessages, _ := strconv.Atoi(endParts[1])
	if numMessages > MaxRetrieveSize {
		numMessages = MaxRetrieveSize
	}
	if numMessages < 0 {
		numMessages = 0
	}

	msgs := store.GetBeforeTime(listener.User.ID, listener.ServerConnection.Name, target, timeFrom, numMessages)

	batchId := makeBatchId()
	listener.Send(nil, "", "BATCH", "+"+batchId, "chathistory", target)

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

func makeBatchId() string {
	length := 8
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	id := fmt.Sprintf("%X", b)
	return id
}
