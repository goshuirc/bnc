// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package main

import (
	"fmt"
	"log"

	"github.com/docopt/docopt-go"
	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/bnc/lib/setup"
)

func main() {
	usage := `GoshuBNC.

GoshuBNC is an IRC bouncer.

Usage:
	bnc initdb [--conf <filename>]
	bnc start [--conf <filename>]
	bnc -h | --help
	bnc --version

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
		ircsetup.InitialSetup(db)
	} else if arguments["start"].(bool) {
		fmt.Println("Starting", ircsetup.CbCyan("GoshuBNC"))

		db := ircbnc.OpenDB(config.Bouncer.DatabasePath)
		bouncer, err := ircbnc.NewBouncer(config, db)
		if err != nil {
			log.Fatal("Could not create bouncer:", err.Error())
		}

		err = bouncer.Run()
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}
