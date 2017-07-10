// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircbnc

import (
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

// nickHandler handles the NICK command.
func nickHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	// always reject dodgy nicknames, makes things immensely easier
	nick, nickError := IrcName(msg.Params[0], false)
	if nickError != nil {
		listener.Send(nil, "", "422", listener.ClientNick, msg.Params[0], "Erroneus nickname")
		return false
	}

	// we ignore NICK messages during registration
	if !listener.Registered {
		listener.ClientNick = nick
		listener.regLocks["NICK"] = true
		return false
	}
	//TODO(dan): Handle NICK messages when connected to servers.
	listener.Send(nil, "", "ERROR", "We're supposed to handle NICK changes here!")
	return true
}

// userHandler handles the USER command.
func userHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	// we ignore the content of USER messages entirely, since we use our internal
	// user and realname when actually connecting to servers
	if !listener.Registered {
		listener.regLocks["USER"] = true
	}
	return false
}

// passHandler handles the PASS command.
func passHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
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

	user, valid := listener.Bouncer.Users[userid]
	if !valid {
		listener.Send(nil, "", "ERROR", "Invalid username or password")
		return true
	}

	loginError := CompareHashAndPassword(user.HashedPassword, listener.Bouncer.Salt, user.Salt, password)

	if loginError == nil {
		listener.User = user
		network, netExists := user.Networks[networkID]
		if netExists {
			network.AddListener(listener)
		}
		return false
	}

	listener.Send(nil, "", "ERROR", "Invalid username or password")
	return true
}

// capHandler handles the CAP command.
func capHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	//TODO(dan): Write CAP handling code.
	return false
}
