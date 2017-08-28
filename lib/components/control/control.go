package bncComponentControl

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

// Nick of the controller
var control_nick string
var control_source string

func Run(manager *ircbnc.Manager) {
	control_nick = manager.StatusNick
	control_source = manager.StatusSource
	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
}

func onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)
	if !event.FromClient {
		return
	}

	msg := event.Message
	listener := event.Listener

	if msg.Command != "PRIVMSG" || msg.Params[0] != control_nick {
		return
	}

	// Stop the message from being sent upstream
	event.Halt = true

	parts := strings.Split(msg.Params[1], " ")
	command := strings.ToLower(parts[0])
	params := parts[1:]

	switch command {
	case "listnetworks":
		commandListNetworks(listener, params, msg)
	case "addnetwork":
		commandAddNetwork(listener, params, msg)
	case "connect":
		commandConnectNetwork(listener, params, msg)
	case "disconnect":
		commandDisconnectNetwork(listener, params, msg)
	}

	// Admin commands
	// TODO: The role apaprently isnt set or saved. do that.
	if listener.User.Role == "Owner" {
		switch command {
		case "adduser":
			commandAddUser(listener, params, msg)
		}
	}
}

func commandAddUser(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) < 2 {
		listener.SendStatus("Usage: adduser [username] [password]")
		return
	}

	manager := listener.Manager
	data := manager.Ds

	newUsername := params[0]
	newPassword := params[1]
	_, exists := manager.Users[newUsername]
	if exists {
		listener.SendStatus("User " + newUsername + " already exists")
		return
	}

	user := ircbnc.NewUser(listener.Manager)
	user.Name = newUsername
	user.Role = "User"
	user.DefaultNick = newUsername
	user.DefaultFbNick = newUsername + "_"
	user.DefaultUser = newUsername
	user.DefaultReal = newUsername
	user.Permissions = []string{"*"}
	data.SetUserPassword(user, newPassword)

	err := data.SaveUser(user)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create user info in database: %s", err.Error()))
		return
	}

	// TODO: This should really be done in DataStore.SaveUser
	manager.Users[user.ID] = user

	listener.SendStatus("User " + newUsername + " added")
}

func commandConnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	netName := listener.ServerConnection.Name
	if len(params) >= 1 {
		netName = params[0]
	}

	net, exists := listener.User.Networks[netName]
	if !exists {
		listener.SendStatus("Network " + netName + " not found")
		return
	}

	net.Connect()
	if net.Foo.Connected {
		listener.SendStatus("Network " + netName + " connected!")
	} else {
		listener.SendStatus("Network " + netName + " could not connect")
	}
}

func commandDisconnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	netName := listener.ServerConnection.Name
	if len(params) >= 1 {
		netName = params[0]
	}

	net, exists := listener.User.Networks[netName]
	if !exists {
		listener.SendStatus("Network " + netName + " not found")
		return
	}

	net.Disconnect()
}

func commandListNetworks(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	table := NewTable()
	table.SetHeader([]string{"Name", "Nick", "Connected", "Address"})

	for _, network := range listener.User.Networks {
		connected := "No"
		network.Foo.RLock()
		if network.Foo.HasRegistered {
			connected = "Yes"
		}
		network.Foo.RUnlock()

		address := network.Addresses[0].Host + ":"
		if network.Addresses[0].UseTLS {
			address += "+"
		}
		address += strconv.Itoa(network.Addresses[0].Port)

		name := network.Name
		if network == listener.ServerConnection {
			name = "*" + name
		}

		table.Append([]string{name, network.Nickname, connected, address})
	}

	table.RenderToListener(listener, control_source, "PRIVMSG")
}

func commandAddNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	sendUsage := func() {
		listener.SendStatus("Usage: addnetwork name address [port] [password]")
		listener.SendStatus("To use SSL/TLS, add + infront of the port number.")
	}

	if len(params) < 2 {
		sendUsage()
		return
	}

	netName := params[0]
	netAddress := params[1]
	netPort := 6667
	netTls := false
	netPassword := ""

	if len(params) >= 3 {
		portParam := params[2]
		if len(portParam) > 2 && portParam[:1] == "+" {
			netTls = true
			portParam = portParam[1:]
		}
		netPort, _ = strconv.Atoi(portParam)
	}

	if len(params) >= 4 {
		netPassword = params[3]
	}

	if netName == "" || netAddress == "" || netPort == 0 {
		sendUsage()
		return
	}

	connection := ircbnc.NewServerConnection()
	connection.User = listener.User
	connection.Name = netName
	connection.Password = netPassword

	newAddress := ircbnc.ServerConnectionAddress{
		Host:      netAddress,
		Port:      netPort,
		UseTLS:    netTls,
		VerifyTLS: false,
	}
	connection.Addresses = append(connection.Addresses, newAddress)
	listener.User.Networks[connection.Name] = connection

	err := listener.Manager.Ds.SaveConnection(connection)
	if err != nil {
		listener.SendStatus("Could not save the new network")
	} else {
		listener.SendStatus("New network saved")
	}
}
