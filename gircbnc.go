// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package main

import (
	"fmt"
	"log"

	"github.com/DanielOaks/gircbnc/ircbnc"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gIRCbnc.

gIRCbnc is an IRC bouncer.

Usage:
	gircbnc initdb [--conf <filename>]
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

	if arguments["initdb"].(bool) {
		ircbnc.InitDB(config.Bouncer.DatabasePath)

		db := ircbnc.OpenDB(config.Bouncer.DatabasePath)
		InitialSetup(db)
	} else if arguments["start"].(bool) {
		fmt.Println("Starting", cbCyan("gIRCbnc"))

		db := ircbnc.OpenDB(config.Bouncer.DatabasePath)
		bouncer, err := ircbnc.NewBouncer(config, db)
		if err != nil {
			log.Fatal("Could not create bouncer:", err.Error())
		}
		bouncer.Run()
	}
}
