// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package main

import (
	"fmt"
	"log"

	"github.com/docopt/docopt-go"
	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/bnc/lib/setup"

	"github.com/goshuirc/bnc/lib/datastores/buntdb"

	// Different parts of the project acting independantly
	"github.com/goshuirc/bnc/lib/components/componentLoader"
)

func main() {
	usage := `GoshuBNC.

GoshuBNC is an IRC bouncer.

Usage:
	bnc init [--conf <filename>]
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

	data := &bncDataStoreBuntdb.DataStore{}
	manager := ircbnc.NewManager(config, data)

	dataErr := data.Init(manager)
	if dataErr != nil {
		log.Fatalln(dataErr.Error())
	}

	if arguments["init"].(bool) {
		setupErr := data.Setup()
		if setupErr != nil {
			log.Fatal("Could not initialise the database: ", err.Error())
		}

		ircsetup.InitialSetup(manager)

	} else if arguments["start"].(bool) {
		fmt.Println("Starting", ircsetup.CbCyan("GoshuBNC"))

		// Start the different components
		bncComponentLoader.Run(manager)

		err = manager.Run()
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}
