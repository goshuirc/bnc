// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package main

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/DanielOaks/gircbnc/ircbnc"
	"github.com/fatih/color"
)

var (
	cbBlue   = color.New(color.Bold, color.FgHiBlue).SprintfFunc()
	cbCyan   = color.New(color.Bold, color.FgHiCyan).SprintfFunc()
	cbYellow = color.New(color.Bold, color.FgHiYellow).SprintfFunc()
	cbRed    = color.New(color.Bold, color.FgHiRed).SprintfFunc()
)

// Section displays a section to the user
func Section(text string) {
	Note("")
	fmt.Println(cbBlue("["), cbYellow("**"), cbBlue("]"), "--", text, "--")
	Note("")
}

// Note displays a note to the user
func Note(text string) {
	fmt.Println(cbBlue("["), cbYellow("**"), cbBlue("]"), text)
}

// Query asks for a value from the user
func Query(prompt string) (string, error) {
	fmt.Print(cbBlue("[ "), cbYellow("??"), cbBlue(" ] "), prompt)

	in := bufio.NewReader(os.Stdin)
	response, err := in.ReadString('\n')
	return strings.TrimRight(response, "\r\n"), err
}

// QueryNoEcho asks for a value from the user without echoing what they type
func QueryNoEcho(prompt string) (string, error) {
	fmt.Print(cbBlue("[ "), cbYellow("??"), cbBlue(" ] "), prompt)

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

// Warn warns the user about something
func Warn(text string) {
	fmt.Println(cbBlue("["), cbRed("**"), cbBlue("]"), text)
}

// Error shows the user an error
func Error(text string) {
	fmt.Println(cbBlue("["), cbRed("!!"), cbBlue("]"), cbRed(text))
}

// InitialSetup performs the initial gircbnc setup
func InitialSetup(db *sql.DB) {
	fmt.Println(cbBlue("["), cbCyan("~~"), cbBlue("]"), "Welcome to", cbCyan("gIRCbnc"))
	Note("We will now run through basic setup.")

	var err error

	// generate the password salt used by the bouncer
	bncSalt, err := ircbnc.NewSalt()
	if err != nil {
		log.Fatal("Could not generate cryptographically-secure salt for the bouncer:", err.Error())
	}

	db.Exec(`INSERT INTO ircbnc (key, value) VALUES ("salt",?)`, base64.StdEncoding.EncodeToString(bncSalt))

	Section("Admin user settings")
	var username string
	var goodUsername string
	for {
		username, err = Query("Username: ")

		if err != nil {
			Error(fmt.Sprintf("Error reading input line: %s", err.Error()))
			continue
		}

		username = strings.TrimSpace(username)

		goodUsername, err = ircbnc.Username(username)
		if err == nil {
			Note(fmt.Sprintf("Username is %s. Will be stored internally as %s.", username, goodUsername))
			break
		} else {
			Error(err.Error())
		}
	}

	// generate our salts
	userSalt, err := ircbnc.NewSalt()
	if err != nil {
		log.Fatal("Could not generate cryptographically-secure salt for the user:", err.Error())
	}

	var passHash []byte
	for {
		password, err := QueryNoEcho("Enter password: ")

		if err != nil {
			Error(fmt.Sprintf("Error reading input line: %s", err.Error()))
			continue
		}

		passwordCompare, err := QueryNoEcho("Confirm password: ")

		if err != nil {
			Error(fmt.Sprintf("Error reading input line: %s", err.Error()))
			continue
		}

		if password != passwordCompare {
			Warn("The supplied passwords do not match")
			continue
		}

		passHash, err = ircbnc.GenerateFromPassword(bncSalt, userSalt, password)

		if err == nil {
			break
		} else {
			Error(fmt.Sprintf("Could not generate password: %s", err.Error()))
			continue
		}
	}

	// get IRC details
	var ircNick string
	for {
		ircNick, err = Query("Enter Nickname: ")
		if err != nil {
			log.Fatal(err.Error())
		}

		ircNick, err = ircbnc.IrcName(ircNick)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	var ircFbNick string
	defaultFallbackNick := fmt.Sprintf("%s_", ircNick)
	for {
		ircFbNick, err = QueryDefault(fmt.Sprintf("Enter Fallback Nickname [%s]: ", defaultFallbackNick), defaultFallbackNick)
		if err != nil {
			log.Fatal(err.Error())
		}

		ircFbNick, err = ircbnc.IrcName(ircFbNick)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	var ircUser string
	for {
		ircUser, err = Query("Enter Username: ")
		if err != nil {
			log.Fatal(err.Error())
		}

		ircUser, err = ircbnc.IrcName(ircUser)
		if err == nil {
			break
		} else {
			Error(err.Error())
		}
	}

	var ircReal string
	ircReal, err = Query("Enter Realname: ")
	if err != nil {
		log.Fatal(err.Error())
	}

	db.Exec(`INSERT INTO users (id, salt, password, default_nickname, default_fallback_nickname, default_username, default_realname) VALUES (?,?,?,?,?,?,?)`,
		goodUsername, userSalt, passHash, ircNick, ircFbNick, ircUser, ircReal)

	Note("User created!")
}
