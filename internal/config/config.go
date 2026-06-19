// SPDX-License-Identifier: BSD-3-Clause

// Package config provides the central configuration of the tool and the CLI options
package config

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GCInterval        time.Duration
	HeartbeatInterval time.Duration
	KeepFor           time.Duration
	ChecksInterval    time.Duration

	DisplayNameAsServiceName bool
	ActiveChecks             bool
	MaxCheckAttempts         float64

	LogLevel         string
	ID               string
	CheckCommand     string
	HeartbeatService string

	StaticServiceVars    map[string]string
	CustomSeverityLevels map[string]string
	MergedSeverityLevels map[string]int

	PluginOutputByStates      bool
	BearerToken               string
	ListenAddr                string
	TLSCertPath               string
	TLSKeyPath                string
	PluginOutputAnnotations   []string
	PluginOutputStateSuffixes []string

	IcingaDisableKeepAlives bool
	IcingaHostname          string
	IcingaUser              string
	IcingaPassword          string
	IcingaTemplates         []string
	IcingaURL               []string
	IcingaTLSConfig         *tls.Config
}

func NewConfigFromCLI(cli *CLI) (*Config, error) {
	conf := Config{
		ID:                       cli.ID,
		ActiveChecks:             cli.ActiveChecks,
		BearerToken:              cli.BearerToken,
		CheckCommand:             cli.CheckCommand,
		ChecksInterval:           cli.ChecksInterval,
		GCInterval:               cli.GCInterval,
		DisplayNameAsServiceName: cli.DisplayNameAsServiceName,
		HeartbeatInterval:        cli.HeartbeatInterval,
		HeartbeatService:         cli.HeartbeatService,
		IcingaDisableKeepAlives:  cli.DisableKeepAlives,
		IcingaHostname:           cli.IcingaHostname,
		IcingaUser:               cli.IcingaUser,
		IcingaPassword:           cli.IcingaPassword,
		IcingaURL:                cli.IcingaURL,
		KeepFor:                  cli.KeepFor,
		ListenAddr:               cli.ListenAddr,
		IcingaTemplates:          cli.Templates,
		MaxCheckAttempts:         float64(cli.MaxCheckAttempts),
		PluginOutputAnnotations:  cli.PluginOutputAnnotations,
		PluginOutputByStates:     cli.PluginOutputByStates,
		StaticServiceVars:        cli.StaticServiceVars,
		MergedSeverityLevels:     mergeSeverityLevels(cli.CustomSeverityLevels),
	}

	// Try to load the TLS configuration
	if cli.TLSCertPath != "" && cli.TLSKeyPath != "" {
		_, errTLS := NewTLSConfig(&TLSConfig{
			KeyFile:  cli.TLSKeyPath,
			CertFile: cli.TLSCertPath,
		})

		if errTLS != nil {
			return &conf, fmt.Errorf("error loading the TLS configuration %w", errTLS)
		}

		conf.TLSKeyPath = cli.TLSKeyPath
		conf.TLSCertPath = cli.TLSCertPath
	}

	conf.IcingaTLSConfig = &tls.Config{
		//nolint: gosec
		InsecureSkipVerify: cli.IcingaInsecureTLS,
	}

	if cli.IcingaCAFile != "" {
		icingaTLS, errIcingaTLS := NewTLSConfig(&TLSConfig{
			InsecureSkipVerify: cli.IcingaInsecureTLS,
			CAFile:             cli.IcingaCAFile,
		})

		if errIcingaTLS != nil {
			return &conf, fmt.Errorf("error loading the Icinga TLS configuration %w", errIcingaTLS)
		}

		conf.IcingaTLSConfig = icingaTLS
	}

	return &conf, nil
}

// MapLogLevel returns the corresponding slog.Level given a string
func MapLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Create the default severity levels and then merge any custom ones into it.
// This keeps the defaults for backwards compatibility and allows both additions and overrides.
func mergeSeverityLevels(customlevels map[string]string) map[string]int {
	allLevels := map[string]int{
		"normal":   0,
		"warning":  1,
		"critical": 2,
	}

	for k, v := range customlevels {
		// Ensure the user set configuration values are valid otherwise default to UNKNOWN
		l, err := strconv.ParseInt(v, 10, 32)
		if err != nil || l < 0 || l > 3 {
			l = 3
		}

		allLevels[strings.ToLower(k)] = int(l)
	}

	return allLevels
}
