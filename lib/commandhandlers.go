// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"strings"

	"log"

	"github.com/goshuirc/irc-go/ircmsg"
)

func loadClientCommands() {
	ClientCommands["NICK"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			// always reject dodgy nicknames, makes things immensely easier
			nick, nickError := IrcName(msg.Params[0], false)
			if nickError != nil {
				listener.Send(nil, "", "422", listener.ClientNick, msg.Params[0], "Erroneus nickname")
				return false
			}

			// we ignore NICK messages during registration
			if !listener.Registered {
				listener.ClientNick = nick
				listener.regLocks.Set("nick", true)
				return false
			}
			//TODO(dan): Handle NICK messages when connected to servers.
			//listener.Send(nil, "", "ERROR", "We're supposed to handle NICK changes here!")
			listener.ServerConnection.Nickname = nick
			return true
		},
	}

	ClientCommands["USER"] = ClientCommand{
		usablePreReg: true,
		minParams:    4,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			// we ignore the content of USER messages entirely, since we use our internal
			// user and realname when actually connecting to servers
			if !listener.Registered {
				listener.regLocks.Set("user", true)
			}
			return false
		},
	}

	ClientCommands["PASS"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			// only accept PASS before registration finishes
			if listener.Registered {
				return false
			}

			splitString := strings.SplitN(msg.Params[0], ":", 2)

			if len(splitString) < 2 {
				listener.Send(nil, "", "ERROR", `Password must be of the format "<username>/<network>:<password>"`)
				return true
			}

			password := splitString[1]

			var userid, networkID string
			if strings.Contains(splitString[0], "/") {
				splitString = strings.Split(splitString[0], "/")
				userid, networkID = splitString[0], splitString[1]
			} else {
				userid = splitString[0]
			}

			authedUserId, authSuccess := listener.Manager.Ds.AuthUser(userid, password)
			if !authSuccess {
				listener.Send(nil, "", "ERROR", "Invalid username or password")
				return true
			}

			user := listener.Manager.Users[authedUserId]
			listener.User = user
			network, netExists := user.Networks[networkID]
			if netExists {
				network.AddListener(listener)
			} else {
				log.Println("Network '" + networkID + "' doesnt exist")
				listener.regLocks.Set("listener", true)
				listener.tryRegistration()
			}

			return false
		},
	}

	ClientCommands["CAP"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			//TODO(dan): Write CAP handling code.
			return false
		},
	}

	ClientCommands["PING"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			// The BNC responds to pings from both the server and client as either
			// could be detached at any point.
			listener.Send(nil, "", "PONG", msg.Params[0])
			return false
		},
	}

	ClientCommands["QUIT"] = ClientCommand{
		usablePreReg: true,
		minParams:    0,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) bool {
			// Just ignore it as clients usually send QUIT when the client is closed
			return false
		},
	}
}
