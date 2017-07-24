// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package main

import (
	"fmt"
	"log"

	"github.com/docopt/docopt-go"
	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/bnc/lib/setup"
	"github.com/tidwall/buntdb"

	// Different parts of the project acting independantly
	"github.com/goshuirc/bnc/lib/components/control"
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

	arguments, _ := docopt.Parse(usage, nil, true, ircbnc.SemVer, false)

	configfile := arguments["--conf"].(string)
	config, err := ircbnc.LoadConfig(configfile)
	if err != nil {
		log.Fatal("Config file did not load successfully:", err.Error())
	}

	if arguments["initdb"].(bool) {
		ircbnc.InitDB(config.Bouncer.DatabasePath)

		db, err := buntdb.Open(config.Bouncer.DatabasePath)
		if err != nil {
			log.Fatal("Could not open DB:", err.Error())
		}
		ircsetup.InitialSetup(db)
	} else if arguments["start"].(bool) {
		fmt.Println("Starting", ircsetup.CbCyan("GoshuBNC"))

		db, err := buntdb.Open(config.Bouncer.DatabasePath)
		if err != nil {
			log.Fatal("Could not open DB:", err.Error())
		}
		manager, err := ircbnc.NewManager(config, db)
		if err != nil {
			log.Fatal("Could not create manager:", err.Error())
		}

		// Start the different components
		bncComponentControl.Run(manager)

		err = manager.Run()
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}
