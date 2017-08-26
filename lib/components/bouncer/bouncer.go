package bncComponentBouncer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

func Run(manager *ircbnc.Manager) {
	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
	manager.Bus.Register(ircbnc.HookNewListenerName, onNewListener)
}

func onNewListener(hook interface{}) {
	event := hook.(*ircbnc.HookNewListener)
	event.Listener.ExtraISupports["BOUNCER"] = "1"
}

func onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)
	if !event.FromClient {
		return
	}

	msg := event.Message
	listener := event.Listener

	if msg.Command != "BOUNCER" {
		return
	}

	// Stop the message from being sent upstream
	event.Halt = true

	command := strings.ToLower(msg.Params[0])
	params := msg.Params[1:]

	switch command {
	case "listnetworks":
		commandListNetworks(listener, params, msg)
	case "addnetwork":
		commandAddNetwork(listener, params, msg)
	case "changenetwork":
		commandChangeNetwork(listener, params, msg)
	case "connect":
		commandConnectNetwork(listener, params, msg)
	case "disconnect":
		commandDisconnectNetwork(listener, params, msg)
	case "listbuffers":
		commandListBuffers(listener, params, msg)
	}
}

func commandConnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) == 0 {
		listener.SendLine("BOUNCER connect * ERR_INVALIDARGS")
	}

	netName := params[0]
	net := getNetworkByName(listener, netName)
	if net == nil {
		listener.SendLine(fmt.Sprintf("BOUNCER connect %s ERR_NETNOTFOUND", netName))
		return
	}

	listener.SendLine(fmt.Sprintf("BOUNCER state %s connecting", netName))
	net.Connect()

	if net.Foo.Connected {
		listener.SendLine(fmt.Sprintf("BOUNCER state connected", netName))
	} else {
		listener.SendLine(fmt.Sprintf("BOUNCER state disconnected", netName))
	}
}

func commandDisconnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) == 0 {
		listener.SendLine("BOUNCER disconnect * ERR_INVALIDARGS")
	}

	netName := params[0]
	net := getNetworkByName(listener, netName)
	if net == nil {
		listener.SendLine(fmt.Sprintf("BOUNCER disconnect %s ERR_NETNOTFOUND", netName))
		return
	}

	net.Disconnect()
	listener.Send(nil, "", "BOUNCER", "state", netName, "disconnected")
}

// [c] bouncer listnetworks
// [s] bouncer listnetworks network=freenode;host=irc.freenode.net;port=6667;state=disconnected;
// [s] bouncer listnetworks network=snoonet;host=irc.snoonet.org;port=6697;state=connected;tls=1
// [s] bouncer listnetworks end
func commandListNetworks(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	for _, network := range listener.User.Networks {
		vals := make(map[string]string)
		vals["network"] = network.Name
		vals["nick"] = network.Nickname
		vals["user"] = network.Username
		vals["host"] = network.Addresses[0].Host
		vals["port"] = strconv.Itoa(network.Addresses[0].Port)
		if network.Addresses[0].UseTLS {
			vals["tls"] = "1"
		} else {
			vals["tls"] = "0"
		}
		if network.Foo.Connected {
			vals["state"] = "connected"
			vals["currentNick"] = network.Foo.Nick
		} else {
			vals["state"] = "disconnected"
		}

		line := ""
		for k, v := range vals {
			line += fmt.Sprintf("%s=%s;", k, v)
		}

		listener.SendLine("BOUNCER listnetworks " + line)
	}

	listener.SendLine("BOUNCER listnetworks end")
}

// [c] bouncer listbuffers <network name>
// [s] bouncer listbuffers freenode network=freenode;buffer=#chan;joined=1;topic=some\stopic
// [s] bouncer listbuffers freenode network=freenode;buffer=somenick;
// [s] bouncer listbuffers freenode end
func commandListBuffers(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) == 0 {
		listener.SendLine("BOUNCER listbuffers * ERR_INVALIDARGS")
	}

	netName := params[0]
	net := getNetworkByName(listener, netName)
	if net == nil {
		listener.SendLine("BOUNCER listbuffers " + netName + " ERR_NETNOTFOUND")
		return
	}

	// TODO: Also list any open query buffers we have
	for _, channel := range net.Channels {
		vals := make(map[string]string)
		vals["network"] = net.Name
		vals["buffer"] = channel.Name
		vals["channel"] = "1"
		// TODO: Store the topic in the channels when we have them
		vals["topic"] = ""
		vals["joined"] = "1"

		line := ""
		for k, v := range vals {
			line += fmt.Sprintf("%s=%s;", k, v)
		}

		listener.Send(nil, "", "BOUNCER", "listbuffers", netName, line)
	}

	listener.SendLine("BOUNCER listbuffers " + net.Name + " end")
}

// [c] bouncer addnetwork network=freenode;host=irc.freenode.net;port=6667;nick=prawnsalad;user=prawn
// [s] bouncer addnetwork ERR_NAMEINUSE freenode
// [s] bouncer addnetwork ERR_NEEDSNAME *
// [s] bouncer addnetwork RPL_OK freenode
func commandAddNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) < 1 {
		listener.SendLine("BOUNCER addnetwork * ERR_INVALIDARGS")
		return
	}

	vars, tagsErr := ircmsg.ParseTags(params[0])
	if tagsErr != nil {
		listener.SendLine("BOUNCER addnetwork * ERR_INVALIDARGS")
		return
	}

	netName := tagValue(vars, "network", "")
	netAddress := tagValue(vars, "host", "")
	netPort, _ := strconv.Atoi(tagValue(vars, "port", "6667"))
	netPassword := tagValue(vars, "password", "")
	netNick := tagValue(vars, "nick", "")
	netUser := tagValue(vars, "user", "")

	netTls := false
	varTls := tagValue(vars, "tls", "0")
	if varTls == "1" {
		netTls = true
	} else {
		netTls = false
	}

	if netName == "" || netAddress == "" || netPort == 0 {
		listener.SendLine("BOUNCER addnetwork * ERR_INVALIDARGS")
		return
	}

	existingNet := getNetworkByName(listener, netName)
	if existingNet != nil {
		listener.SendLine("BOUNCER addnetwork " + existingNet.Name + " ERR_NAMEINUSE ")
		return
	}

	connection := ircbnc.NewServerConnection()
	connection.User = listener.User
	connection.Name = netName
	connection.Password = netPassword

	if netNick != "" {
		connection.Nickname = netNick
	} else {
		connection.Nickname = listener.User.Name
	}
	if netUser != "" {
		connection.Username = netUser
	} else {
		connection.Username = listener.User.Name
	}

	newAddress := ircbnc.ServerConnectionAddress{
		Host:      netAddress,
		Port:      netPort,
		UseTLS:    netTls,
		VerifyTLS: false,
	}
	connection.Addresses = append(connection.Addresses, newAddress)
	listener.User.Networks[connection.Name] = connection

	saveErr := listener.Manager.Ds.SaveConnection(connection)
	if saveErr != nil {
		listener.SendLine("BOUNCER addnetwork " + netName + " ERR_UNKNOWN :Error saving the network")
	} else {
		listener.SendLine("BOUNCER addnetwork " + netName + " RPL_OK")
	}
}

// [c] bouncer changenetwork freenode host=irc.freenode.net;port=6667;
// [s] bouncer changenetwork RPL_OK freenode
func commandChangeNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	if len(params) < 2 {
		listener.SendLine("BOUNCER changenetwork * ERR_INVALIDARGS")
		return
	}

	vars, tagsErr := ircmsg.ParseTags(params[1])
	if tagsErr != nil {
		listener.SendLine("BOUNCER changenetwork * ERR_INVALIDARGS")
		return
	}

	netName := params[0]
	net := getNetworkByName(listener, netName)
	if net == nil {
		listener.SendLine("BOUNCER changenetwork * ERR_NETNOTFOUND")
		return
	}

	netAddress := tagValue(vars, "host", "")
	if netAddress != "" {
		net.Addresses[0].Host = netAddress
	}

	netPort, _ := strconv.Atoi(tagValue(vars, "network", "6667"))
	if netPort > 0 {
		net.Addresses[0].Port = netPort
	}

	// Using the default of . is so hacky. But I'm tired, and this is easier for now. ~Darren
	netPassword := tagValue(vars, "password", ".")
	if netPassword != "." {
		net.Password = netPassword
	}

	netNick := tagValue(vars, "nick", "")
	if netNick != "" {
		net.Nickname = netNick
	}

	netUser := tagValue(vars, "user", "")
	if netUser != "" {
		net.Username = netUser
	}

	netTls := tagValue(vars, "tls", "0")
	if netTls == "1" {
		net.Addresses[0].UseTLS = true
	} else if netTls == "0" {
		net.Addresses[0].UseTLS = false
	}
	saveErr := listener.Manager.Ds.SaveConnection(net)
	if saveErr != nil {
		listener.SendLine("BOUNCER changenetwork " + net.Name + " ERR_UNKNOWN :Error saving the network")
	} else {
		listener.SendLine("BOUNCER changenetwork " + net.Name + " RPL_OK")
	}
}

func tagValue(tags map[string]ircmsg.TagValue, name string, def string) string {
	val, exists := tags[name]
	if !exists {
		return def
	}

	if !val.HasValue {
		return ""
	}

	return val.Value
}

func getNetworkByName(listener *ircbnc.Listener, netName string) *ircbnc.ServerConnection {
	for _, network := range listener.User.Networks {
		if strings.ToLower(network.Name) == strings.ToLower(netName) {
			return network
		}
	}

	return nil
}
