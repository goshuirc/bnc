// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package ircsetup

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/fatih/color"
	"github.com/goshuirc/bnc/lib"
)

var (
	CbBlue   = color.New(color.Bold, color.FgHiBlue).SprintfFunc()
	CbCyan   = color.New(color.Bold, color.FgHiCyan).SprintfFunc()
	CbYellow = color.New(color.Bold, color.FgHiYellow).SprintfFunc()
	CbRed    = color.New(color.Bold, color.FgHiRed).SprintfFunc()
)

// Section displays a section to the user
func Section(text string) {
	Note("")
	fmt.Println(CbBlue("["), CbYellow("**"), CbBlue("]"), "--", text, "--")
	Note("")
}

// Note displays a note to the user
func Note(text string) {
	fmt.Println(CbBlue("["), CbYellow("**"), CbBlue("]"), text)
}

// Query asks for a value from the user
func Query(prompt string) (string, error) {
	fmt.Print(CbBlue("[ "), CbYellow("??"), CbBlue(" ] "), prompt)

	in := bufio.NewReader(os.Stdin)
	response, err := in.ReadString('\n')
	return strings.TrimRight(response, "\r\n"), err
}

// QueryNoEcho asks for a value from the user without echoing what they type
func QueryNoEcho(prompt string) (string, error) {
	fmt.Print(CbBlue("[ "), CbYellow("??"), CbBlue(" ] "), prompt)

	response, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Print("\n")
	return string(response), err
}

// QueryDefault asks for a value, falling back to a default
func QueryDefault(prompt string, defaultValue string) (string, error) {
	response, err := Query(prompt)

	if err != nil {
		return "", err
	}

	if len(strings.TrimSpace(response)) < 1 {
		return defaultValue, nil
	}
	return response, nil
}

// QueryBool asks for a true/false value from the user
func QueryBool(prompt string) (bool, error) {
	for {
		response, err := Query(prompt)
		if err != nil {
			return false, err
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if len(response) < 1 {
			continue
		}

		// check for yes/true/1 or no/false/0
		if strings.Contains("yt1", string(response[0])) {
			return true, nil
		} else if strings.Contains("nf0", string(response[0])) {
			return false, nil
		}
	}
}

// Warn warns the user about something
func Warn(text string) {
	fmt.Println(CbBlue("["), CbRed("**"), CbBlue("]"), text)
}

// Error shows the user an error
func Error(text string) {
	fmt.Println(CbBlue("["), CbRed("!!"), CbBlue("]"), CbRed(text))
}

// InitialSetup performs the initial GoshuBNC setup
func InitialSetup(manager *ircbnc.Manager) {
	fmt.Println(CbBlue("["), CbCyan("~~"), CbBlue("]"), "Welcome to", CbCyan("GoshuBNC"))
	Note("We will now run through basic setup.")

	data := manager.Ds
	var err error

	data.Setup()

	Section("Admin user settings")
	var username string
	var password string

	var ircNick string
	var ircFbNick string // fallback nick
	var ircUser string
	var ircReal string

	for {
		inputUsername, err := Query("Username: ")
		if err != nil {
			log.Fatal(err.Error())
		}

		inputUsername = strings.TrimSpace(inputUsername)
		inputUsername, err = ircbnc.BncName(inputUsername)
		if err == nil {
			username = inputUsername
			Note(fmt.Sprintf("Username is %s.", username))
			break
		} else {
			Error(err.Error())
		}
	}

	// Read the password
	for {
		newPassword, err := QueryNoEcho("Enter password: ")

		if err != nil {
			Error(fmt.Sprintf("Error reading input line: %s", err.Error()))
			continue
		}

		passwordCompare, err := QueryNoEcho("Confirm password: ")

		if err != nil {
			Error(fmt.Sprintf("Error reading input line: %s", err.Error()))
			continue
		}

		if newPassword != passwordCompare {
			Warn("The supplied passwords do not match")
			continue
		}

		password = newPassword
		break
	}

	// Get IRC details
	for {
		ircNick, err = QueryDefault(fmt.Sprintf("Enter Nickname [%s]: ", username), username)
		if err != nil {
			log.Fatal(err.Error())
		}

		ircNick, err = ircbnc.IrcName(ircNick, false)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	defaultFallbackNick := fmt.Sprintf("%s_", ircNick)
	for {
		ircFbNick, err = QueryDefault(fmt.Sprintf("Enter Fallback Nickname [%s]: ", defaultFallbackNick), defaultFallbackNick)
		if err != nil {
			log.Fatal(err.Error())
		}

		ircFbNick, err = ircbnc.IrcName(ircFbNick, false)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	for {
		ircUser, err = QueryDefault(fmt.Sprintf("Enter Username [%s]: ", username), username)
		if err != nil {
			log.Fatal(err.Error())
		}

		ircUser, err = ircbnc.IrcName(ircUser, false)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	ircReal, err = Query("Enter Realname: ")
	if err != nil {
		log.Fatal(err.Error())
	}

	user := ircbnc.NewUser(manager)
	user.Name = username
	user.Role = "Owner"
	user.DefaultNick = ircNick
	user.DefaultFbNick = ircFbNick
	user.DefaultUser = ircUser
	user.DefaultReal = ircReal
	user.Permissions = []string{"*"}
	data.SetUserPassword(user, password)

	err = data.SaveUser(user)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create user info in database: %s", err.Error()))
		return
	}

	// now setup default networks for that user
	Section("Network Setup")

	for {
		setupNewNet, err := QueryBool("Set up a network? (y/n) ")
		if err != nil {
			log.Fatal(err.Error())
		}

		if !setupNewNet {
			break
		}

		var goodNetName string
		for {
			netName, err := Query("Name (e.g. freenode): ")
			if err != nil {
				log.Fatal(err.Error())
			}

			goodNetName, err = ircbnc.BncName(netName)
			if err == nil {
				Note(fmt.Sprintf("Network name is %s. Will be stored internally as %s.", netName, goodNetName))
				break
			} else {
				Error(err.Error())
			}
		}

		var serverAddress string
		for {
			serverAddress, err = Query("Server host (e.g. chat.freenode.net): ")
			if err != nil {
				log.Fatal(err.Error())
			}

			if len(strings.TrimSpace(serverAddress)) < 1 {
				Error("Hostname must have at least one character!")
				continue
			}

			break
		}

		serverUseTLS, err := QueryBool("Server uses SSL/TLS? (y/n) ")
		if err != nil {
			log.Fatal(err.Error())
		}

		var serverVerifyTLS bool
		if serverUseTLS {
			serverVerifyTLS, err = QueryBool("Verify SSL/TLS certificates? (y/n) ")
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		var defaultPort int
		if serverUseTLS {
			defaultPort = 6697
		} else {
			defaultPort = 6667
		}

		var serverPort int
		for {
			portString, err := QueryDefault(fmt.Sprintf("Server Port [%d]: ", defaultPort), strconv.Itoa(defaultPort))
			if err != nil {
				log.Fatal(err.Error())
			}

			serverPort, err = strconv.Atoi(portString)
			if err != nil {
				Error(err.Error())
				continue
			}

			if (serverPort < 1) || (serverPort > 65535) {
				Error("Port number can be 1 - 65535")
				continue
			}

			break
		}

		serverPass, err := Query("Server connection password (probably empty): ")
		if err != nil {
			log.Fatal(err.Error())
		}

		var serverChannels ircbnc.ServerConnectionBuffers
		for {
			serverChannelsString, err := Query("Channels to autojoin (separated by spaces): ")
			if err != nil {
				log.Fatal(err.Error())
			}

			for _, channel := range strings.Fields(serverChannelsString) {
				channel, err := ircbnc.IrcName(channel, true)
				if err != nil {
					break
				}

				serverChannels.Add(&ircbnc.ServerConnectionBuffer{
					Channel: true,
					Name:    channel,
				})
			}

			if err != nil {
				Error(err.Error())
				continue
			}

			break
		}

		connection := ircbnc.NewServerConnection()
		connection.User = user
		connection.Name = goodNetName
		connection.Password = serverPass
		newAddress := ircbnc.ServerConnectionAddress{
			Host:      serverAddress,
			Port:      serverPort,
			UseTLS:    serverUseTLS,
			VerifyTLS: serverVerifyTLS,
		}
		connection.Addresses = append(connection.Addresses, newAddress)
		for _, channel := range serverChannels {
			connection.Buffers[channel.Name] = channel
		}

		err = connection.Save()
		if err != nil {
			log.Fatal(fmt.Sprintf("Could not create server connection [%s] in database: %s", goodNetName, err.Error()))
			return
		}
	}

	fmt.Println(CbBlue("["), CbCyan("~~"), CbBlue("]"), CbCyan("GoshuBNC"), "is now configured!")
	Note("You can now launch GoshuBNC and connect to it with your IRC client")
}
