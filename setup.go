// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

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
	return response, err
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
func InitialSetup(*sql.DB) {
	fmt.Println(cbBlue("["), cbCyan("~~"), cbBlue("]"), "Welcome to", cbCyan("gIRCbnc"))
	Note("We will now run through basic setup.")

	Section("Admin user settings")
	var username string
	var goodUsername string
	var err error
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
}
