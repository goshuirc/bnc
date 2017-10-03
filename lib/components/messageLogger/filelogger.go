package bncComponentLogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

type FileMessageDatastore struct {
	logPath string
}

func NewFileMessageDatastore(config map[string]string) *FileMessageDatastore {
	ds := &FileMessageDatastore{}

	ds.logPath = config["path"]
	if !strings.HasSuffix(ds.logPath, "/") {
		ds.logPath += "/"
	}

	return ds
}

func (ds *FileMessageDatastore) SupportsStore() bool {
	return true
}
func (ds *FileMessageDatastore) SupportsRetrieve() bool {
	return false
}
func (ds *FileMessageDatastore) SupportsSearch() bool {
	return false
}

func (ds *FileMessageDatastore) Store(event *ircbnc.HookIrcRaw) {
	if ds.logPath == "" {
		return
	}

	line, destination := createLineFromMessage(event)
	if line == "" || destination == "" {
		return
	}

	// Make sure the chat directly exists
	logPath := filepath.Join(ds.logPath, event.User.ID, event.Server.Name)
	os.MkdirAll(logPath, os.ModePerm)
	filename := filepath.Join(logPath, destination+".log")

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		println(err.Error())
		return
	}

	f.WriteString(line + "\n")
	f.Close()
}
func (ds *FileMessageDatastore) GetFromTime(string, string, string, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
func (ds *FileMessageDatastore) GetBeforeTime(string, string, string, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}
func (ds *FileMessageDatastore) Search(string, string, string, time.Time, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}

func createLineFromMessage(event *ircbnc.HookIrcRaw) (string, string) {
	line := ""
	destination := ""

	message := event.Message

	if event.FromServer {
		switch message.Command {
		case "PRIVMSG":
			line = fmt.Sprintf("<%s> %s", message.Prefix, message.Params[1])
			destination = message.Params[0]
		case "NOTICE":
			// TODO: Whats the norm format for logging notices?
			line = fmt.Sprintf("<%s> [NOTICE] %s", message.Prefix, message.Params[1])
			destination = message.Params[0]
		case "JOIN":
			line = fmt.Sprintf("* %s has joined %s", message.Prefix, message.Params[0])
			destination = message.Params[0]
		case "PART":
			line = fmt.Sprintf("* %s has left %s", message.Prefix, message.Params[0])
			destination = message.Params[0]
		case "QUIT":
			// line = fmt.Sprintf("* %s has quit", message.Prefix)
			// destination = ?
			// TODO: ^ needs to log into all its channels
		case "KICK":
			line = fmt.Sprintf(
				"* %s has been kicked from %s by %s (%s)",
				message.Params[1],
				message.Params[0],
				message.Prefix,
				message.Params[2],
			)
			destination = message.Params[0]
		}
	} else if event.FromClient && event.Listener.ServerConnection != nil {
		switch message.Command {
		case "PRIVMSG":
			currentNick := event.Listener.ServerConnection.Nickname
			line = fmt.Sprintf("<%s> %s", currentNick, message.Params[1])
			destination = message.Params[0]
		case "NOTICE":
			currentNick := event.Listener.ServerConnection.Nickname
			// TODO: Whats the norm format for logging notices?
			line = fmt.Sprintf("<%s> %s", currentNick, message.Params[1])
			destination = message.Params[0]
		}
	}

	if line != "" {
		line = fmt.Sprintf("[%s] %s", time.Now(), line)
	}
	return line, destination
}
