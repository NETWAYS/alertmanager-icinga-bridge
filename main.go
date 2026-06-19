// SPDX-License-Identifier: BSD-3-Clause

// Package main parses the CLI flags and starts the various components
package main

import (
	"context"
	"fmt"
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

var (
	// These get filled at build time with the proper values.
	version = "development"
	commit  = "HEAD"
	date    = "latest"
)

// buildVersion creates a string that contains the executable's version
func buildVersion() string {
	result := version

	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}

	if date != "" {
		result = fmt.Sprintf("%s\ndate: %s", result, date)
	}

	return result
}

func main() {
	var cli config.CLI
	// Create and parse CLI flags
	kong.Parse(&cli,
		kong.Name("alertmanager-icinga-bridge"),
		kong.Description(`The Alertmanager to Icinga bridge can receive alerts from the Prometheus Alertmanager's generic webhook receiver and creates Icinga Services for these alerts.`),
		kong.Vars{"version": buildVersion()},
	)

	// Central logger that we pass to the components
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.MapLogLevel(cli.Loglevel)}))
	slog.SetDefault(logger)

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
