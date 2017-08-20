package bncComponentLogger

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"

	_ "github.com/mattn/go-sqlite3"
)

const TYPE_MESSAGE = 1
const TYPE_ACTION = 2
const TYPE_NOTICE = 3

type SqliteMessage struct {
	ts          int32
	user        string
	network     string
	buffer      string
	from        string
	messageType int
	line        string
}

type SqliteMessageDatastore struct {
	dbPath       string
	db           *sql.DB
	messageQueue chan SqliteMessage
}

func (ds *SqliteMessageDatastore) SupportsStore() bool {
	return true
}
func (ds *SqliteMessageDatastore) SupportsRetrieve() bool {
	return true
}
func (ds *SqliteMessageDatastore) SupportsSearch() bool {
	return false
}
func NewSqliteMessageDatastore(config map[string]string) SqliteMessageDatastore {
	ds := SqliteMessageDatastore{}

	ds.dbPath = config["database"]
	db, err := sql.Open("sqlite3", ds.dbPath)
	if err != nil {
		log.Fatal(err)
	}

	ds.db = db

	// Create the tables if needed
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS messages (uid TEXT, netid TEXT, ts INT, buffer TEXT, fromNick TEXT, type INT, line TEXT)")
	if err != nil {
		log.Fatal("Error creates messages sqlite database:", err.Error())
	}

	// Start the queue to insert messages
	ds.messageQueue = make(chan SqliteMessage)
	go ds.messageWriter()

	return ds
}

func (ds SqliteMessageDatastore) messageWriter() {
	storeStmt, err := ds.db.Prepare("INSERT INTO messages (uid, netid, ts, buffer, fromNick, type, line) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err.Error())
	}
	for {
		message, isOK := <-ds.messageQueue
		if !isOK {
			break
		}

		storeStmt.Exec(
			message.user,
			message.network,
			message.ts,
			message.buffer,
			message.from,
			message.messageType,
			message.line,
		)
	}
}

func (ds SqliteMessageDatastore) Store(event *ircbnc.HookIrcRaw) {
	from, buffer, messageType, line := extractMessageParts(event)
	if line == "" {
		return
	}

	ds.messageQueue <- SqliteMessage{
		ts:          int32(time.Now().UTC().Unix()),
		user:        event.User.ID,
		network:     event.Server.Name,
		buffer:      buffer,
		from:        from,
		messageType: messageType,
		line:        line,
	}
}
func (ds SqliteMessageDatastore) GetFromTime(userID string, networkID string, buffer string, from time.Time, num int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
func (ds SqliteMessageDatastore) GetBeforeTime(userID string, networkID string, buffer string, from time.Time, num int) []*ircmsg.IrcMessage {
	messages := []*ircmsg.IrcMessage{}

	sql := "SELECT ts, fromNick, type, line FROM messages WHERE uid = ? AND netid = ? AND buffer = ? AND ts < ? ORDER BY ts LIMIT ?"
	rows, err := ds.db.Query(sql, userID, networkID, strings.ToLower(buffer), int32(from.UTC().Unix()), num)
	if err != nil {
		log.Println("GetBeforeTime() error: " + err.Error())
		return messages
	}
	for rows.Next() {
		var ts int32
		var from string
		var messageType int
		var line string
		rows.Scan(&ts, &from, &messageType, &line)

		v := ircmsg.TagValue{}
		v.Value = time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
		v.HasValue = true
		mTags := make(map[string]ircmsg.TagValue)
		mTags["server-time"] = v

		mPrefix := from
		mCommand := "PRIVMSG"
		mParams := []string{
			buffer,
			line,
		}

		if messageType == TYPE_ACTION {
			mParams[1] = "\x01" + mParams[1]
		} else if messageType == TYPE_NOTICE {
			mCommand = "NOTICE"
		}

		m := ircmsg.MakeMessage(&mTags, mPrefix, mCommand, mParams...)
		messages = append(messages, &m)
	}

	// TODO: Private messages should be stored with the buffer name as the other user.
	return messages
}
func (ds SqliteMessageDatastore) Search(string, string, string, time.Time, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}

func extractMessageParts(event *ircbnc.HookIrcRaw) (string, string, int, string) {
	messageType := TYPE_MESSAGE
	from := ""
	buffer := ""
	line := ""

	message := event.Message

	if event.FromServer {
		switch message.Command {
		case "PRIVMSG":
			line = message.Params[1]
			if strings.HasPrefix(line, "\x01ACTION") {
				messageType = TYPE_ACTION
				line = line[1:]
			} else if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_MESSAGE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			// TODO: Extract the nick from the prefix
			from = message.Prefix

		case "NOTICE":
			line = message.Params[1]
			if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_NOTICE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			// TODO: Extract the nick from the prefix
			from = message.Prefix
		}
	} else if event.FromClient {
		switch message.Command {
		case "PRIVMSG":
			line = message.Params[1]
			if strings.HasPrefix(line, "\x01ACTION") {
				messageType = TYPE_ACTION
				line = line[1:]
			} else if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_MESSAGE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			from = event.Listener.ServerConnection.Nickname

		case "NOTICE":
			line = message.Params[1]
			if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_NOTICE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			from = event.Listener.ServerConnection.Nickname
		}
	}

	from = strings.ToLower(from)
	buffer = strings.ToLower(buffer)

	return from, buffer, messageType, line
}
