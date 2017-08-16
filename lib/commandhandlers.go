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
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// always reject dodgy nicknames, makes things immensely easier
			nick, nickError := IrcName(msg.Params[0], false)
			if nickError != nil {
				listener.Send(nil, "", "422", listener.ClientNick, msg.Params[0], "Erroneus nickname")
				return
			}

			// we ignore NICK messages during registration
			if !listener.Registered {
				listener.ClientNick = nick
				listener.regLocks.Set("nick", true)
				return
			}
			//TODO(dan): Handle NICK messages when connected to servers.
			//listener.Send(nil, "", "ERROR", "We're supposed to handle NICK changes here!")
			listener.ServerConnection.Nickname = nick
		},
	}

	ClientCommands["USER"] = ClientCommand{
		usablePreReg: true,
		minParams:    4,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// we ignore the content of USER messages entirely, since we use our internal
			// user and realname when actually connecting to servers
			if !listener.Registered {
				listener.regLocks.Set("user", true)
			}
		},
	}

	ClientCommands["PASS"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// only accept PASS before registration finishes
			if listener.Registered {
				return
			}

			splitString := strings.SplitN(msg.Params[0], ":", 2)

			if len(splitString) < 2 {
				listener.Send(nil, "", "ERROR", `Password must be of the format "<username>/<network>:<password>"`)
				return
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
				return
			}

			user := listener.Manager.Users[authedUserId]
			listener.User = user
			network, netExists := user.Networks[networkID]
			if netExists {
				network.AddListener(listener)
			} else {
				log.Println("Network '" + networkID + "' doesnt exist")
			}

			listener.regLocks.Set("pass", true)
		},
	}

	ClientCommands["CAP"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// We're starting CAP negotiations so don't complete regisration until then
			listener.regLocks.Set("cap", false)

			availableCaps := map[string]string{
				"account-notify": "",
			}

			command := getParam(&msg, 0)
			if command == "LS" {
				capList := ""
				for cap, val := range availableCaps {
					capList += cap
					if val != "" {
						capList += "=" + val
					}
					capList += " "
				}
				listener.Send(nil, "", "CAP", "*", "LS", capList)
			} else if command == "REQ" {
				caps := strings.Split(getParam(&msg, 1), " ")
				acked := []string{}
				for _, cap := range caps {
					capVal, isAvailable := availableCaps[cap]
					if isAvailable {
						listener.Caps[cap] = capVal
						acked = append(acked, cap)
					}
				}

				listener.Send(nil, "", "CAP", "*", "ACK", strings.Join(acked, " "))
			} else if command == "END" {
				listener.regLocks.Set("cap", true)
			}
		},
	}

	ClientCommands["PING"] = ClientCommand{
		usablePreReg: true,
		minParams:    1,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// The BNC responds to pings from both the server and client as either
			// could be detached at any point.
			listener.Send(nil, "", "PONG", msg.Params[0])
		},
	}

	ClientCommands["QUIT"] = ClientCommand{
		usablePreReg: true,
		minParams:    0,
		handler: func(listener *Listener, msg ircmsg.IrcMessage) {
			// Just ignore it as clients usually send QUIT when the client is closed
		},
	}
}

func getParam(msg *ircmsg.IrcMessage, idx int) string {
	if len(msg.Params)-1 < idx {
		return ""
	}

	return msg.Params[idx]
}
