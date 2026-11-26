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
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/hosts"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"

	"github.com/alecthomas/kong"
)

var (
	// These get filled at build time with the proper values.
	version = "development"
	commit  = "HEAD"
)

// buildVersion creates a string that contains the executable's version
func buildVersion() string {
	result := version

	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cli.Loglevel}))
	slog.SetDefault(logger)

	cfg, errConfig := config.NewConfigFromCLI(&cli)

	if errConfig != nil {
		logger.Error("Could not load configuration", "component", "main", "error", errConfig.Error())
		os.Exit(1)
	}

	// Derive hosts from alert labels and create them on demand
	resolver, errResolver := hosts.NewResolver(cfg.HostLabels, cfg.HostLabelSeparator, cfg.HostNamePrefix, cfg.HostMappingFile)
	if errResolver != nil {
		logger.Error("Could not load host mapping", "component", "main", "error", errResolver.Error())
		os.Exit(1)
	}

	// Create a central context to propagate a shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create Icinga Client
	icingaClient := icinga2.NewClient(cfg, logger)

	logger.Info("Starting alertmanager-icinga-bridge", "version", version, "commit", commit, "component", "main")

	hostManager := hosts.NewManager(hosts.ManagerConfig{
		ID:           cfg.ID,
		Templates:    cfg.HostTemplates,
		CheckCommand: cfg.HostCheckCommand,
		Zone:         cfg.HostZone,
		StaticVars:   cfg.StaticHostVars,
		CacheTTL:     cfg.HostCacheTTL,
	}, icingaClient, logger)

	if resolver.Enabled() {
		logger.Info("Deriving hosts from alert labels", "component", "main", "labels", cfg.HostLabels, "mapping", cfg.HostMappingFile)
		go reloadOnHangup(ctx, logger, resolver)
	}

	// Create and start the Service Garbage Collector
	garbagecol := gc.NewGarbageCollector(cfg, logger, icingaClient).WithHostCache(hostManager)
	garbagecol.Start(ctx)

	// Create and start the API Listener
	listener := api.NewListener(cfg, logger, icingaClient).WithHosts(resolver, hostManager)

	//nolint: noinlineerr
	if err := listener.Run(ctx); err != nil {
		logger.Error("Listener has finished with an error", "component", "main", "error", err.Error())
	} else {
		logger.Info("Listener has finished", "component", "main")
	}
}

// reloadOnHangup re-reads the host mapping file whenever the process receives SIGHUP
func reloadOnHangup(ctx context.Context, logger *slog.Logger, resolver *hosts.Resolver) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	defer signal.Stop(hup)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			if resolver.Path() == "" {
				logger.Info("Received SIGHUP, but no host mapping file is configured", "component", "main")
				continue
			}

			errReload := resolver.Reload()
			if errReload != nil {
				logger.Error("Could not reload host mapping, keeping the previous one", "component", "main", "error", errReload.Error())
				continue
			}

			logger.Info("Reloaded host mapping", "component", "main", "file", resolver.Path())
		}
	}
}
