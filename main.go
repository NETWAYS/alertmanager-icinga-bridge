// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/api"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/gc"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"

	"github.com/alecthomas/kong"
)

// nolint: gochecknoglobals
var (
	// These get filled at build time with the proper values.
	version = "development"
	commit  = "HEAD"
	date    = "latest"
)

func main() {
	var cli config.CLI
	// Create and parse CLI flags -> move to kong
	kong.Parse(&cli,
		kong.Name("alertmanager-icinga-bridge"),
		kong.Description("alertmanager-icinga-bridge takes in Alertmanager alerts through a webhook, translates them into Icinga2 services and posts them using the Icinga API"),
		kong.Vars{"version": version},
	)

	// Central logger that we pass to the components
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.MapLogLevel(cli.Loglevel)}))

	cfg, errConfig := config.NewConfigFromCLI(&cli)

	if errConfig != nil {
		logger.Error("Could not load configuration", "component", "main", "error", errConfig.Error())
		os.Exit(1)
	}

	// Create a central context to propagate a shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create Icinga Client
	icingaClient := icinga2.NewClient(cfg, logger)

	logger.Info("Starting alertmanager-icinga-brigde", "version", version, "commit", commit, "date", date, "component", "main")

	// Create and start the Service Garbage Collector
	garbagecol := gc.NewGarbageCollector(cfg, logger, icingaClient)
	garbagecol.Start(ctx)

	// Create and start the API Listener
	//nolint: noinlineerr
	if err := api.NewListener(cfg, logger, icingaClient).Run(ctx); err != nil {
		logger.Error("Listener has finished with an error", "component", "main", "error", err.Error())
	} else {
		logger.Info("Listener has finished", "component", "main")
	}
}
