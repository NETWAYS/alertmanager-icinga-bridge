// Licensed under "BSD 3-Clause". See LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/api"

	"github.com/alecthomas/kingpin/v2"
)

// Version information
var (
	Version   = "undefined"
	BuildDate = "Now"
)

func main() {
	app := kingpin.New("alertmanager-icinga-bridge", "alertmanager-icinga-bridge takes in Alertmanager alerts through a webhook, translates them into Icinga2 services and posts them to Icinga using the Icinga API").Version(Version)
	api.ConfigureServeCommand(app)

	fmt.Printf("alertmanager-icinga-bridge %v\n", Version)
	fmt.Printf("Build time: %v\n\n", BuildDate)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
