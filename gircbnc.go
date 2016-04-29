// written by Daniel Oaks <daniel@danieloaks.net>
// released under the CC0 Public Domain license

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/DanielOaks/gircbnc/ircbnc"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gIRCbnc.

gIRCbnc is an IRC bouncer.

Usage:
	gircbnc start [--conf <filename>]
	gircbnc -h | --help
	gircbnc --version

Options:
	--conf <filename>  Configuration file to use [default: bnc.yaml].
	-h --help          Show this screen.
	--version          Show version.`

	arguments, _ := docopt.Parse(usage, nil, true, ircbnc.Version(), false)

	configfile := arguments["--conf"].(string)
	config, err := ircbnc.LoadConfig(configfile)
	if err != nil {
		log.Fatal("Config file did not load successfully:", err.Error())
	}

	if arguments["start"].(bool) {
		fmt.Println("Starting gIRCbnc")

		var err error
		if err != nil {
			fmt.Println("Connection error:", err)
			os.Exit(1)
		}

		// start
		fmt.Println(config.Bouncer.Listeners)
	}
}
