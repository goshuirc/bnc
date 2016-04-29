// written by Daniel Oaks <daniel@danieloaks.net>
// released under the CC0 Public Domain license

package main

import (
	"fmt"
	"os"

	"github.com/DanielOaks/gircbnc/ircbnc"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gIRCbnc.

Usage:
	gircbnc start
	gircbnc -h | --help
	gircbnc --version

Options:
	-h --help  Show this screen.
	--version  Show version.`

	arguments, _ := docopt.Parse(usage, nil, true, ircbnc.Version(), false)

	if arguments["start"].(bool) {
		fmt.Println("Starting gIRCbnc")

		var err error
		if err != nil {
			fmt.Println("Connection error:", err)
			os.Exit(1)
		}

		// start
	}
}
