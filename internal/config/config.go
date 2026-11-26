// SPDX-License-Identifier: BSD-3-Clause

// Package config provides the central configuration of the tool and the CLI options
package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
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
	AlertFingerprintExcludes []string

	LogLevel         string
	ID               string
	CheckCommand     string
	HeartbeatService string

	StaticServiceVars    map[string]string
	CustomSeverityLevels map[string]StatusCode

	PluginOutputByStates bool
	BearerTokens         []string
	ListenAddr           string
	TLSCertPath          string
	TLSKeyPath           string

	PluginOutputAnnotations  []string
	IcingaHostObjectLabels   []string
	IcingaHostCheckLabels    []string
	IcingaHostZoneLabels     []string
	IcingaHostTemplateLabels []string
	NotesTextAnnotations     []string
	NotesURLAnnotations      []string

	IcingaDisableKeepAlives bool
	IcingaHostname          string
	IcingaUser              string
	IcingaPassword          string
	IcingaTemplates         []string
	IcingaURL               []string
	IcingaTLSConfig         *tls.Config

	HostLabels         []string
	HostLabelSeparator string
	HostNamePrefix     string
	HostMappingFile    string
	HostTemplates      []string
	HostCheckCommand   string
	HostZone           string
	StaticHostVars     map[string]string
	HostCacheTTL       time.Duration
	HostKeepFor        time.Duration
}

func NewConfigFromCLI(cli *CLI) (*Config, error) {
	conf := Config{
		ID:                       cli.ID,
		ActiveChecks:             cli.ActiveChecks,
		BearerTokens:             cli.BearerToken,
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
		AlertFingerprintExcludes: cli.AlertFingerprintExcludes,
		PluginOutputByStates:     cli.PluginOutputByStates,
		PluginOutputAnnotations:  cli.PluginOutputAnnotations,
		IcingaHostObjectLabels:   cli.IcingaHostObjectLabels,
		IcingaHostCheckLabels:    cli.IcingaHostCheckLabels,
		IcingaHostZoneLabels:     cli.IcingaHostZoneLabels,
		IcingaHostTemplateLabels: cli.IcingaHostTemplateLabels,
		NotesTextAnnotations:     cli.NotesTextAnnotations,
		NotesURLAnnotations:      cli.NotesURLAnnotations,
		StaticServiceVars:        cli.StaticServiceVars,
		CustomSeverityLevels:     cli.CustomSeverityLevels,
		HostLabels:               nonEmpty(cli.HostLabels),
		HostLabelSeparator:       cli.HostLabelSeparator,
		HostNamePrefix:           cli.HostNamePrefix,
		HostMappingFile:          cli.HostMappingFile,
		HostTemplates:            nonEmpty(cli.HostTemplates),
		HostCheckCommand:         cli.HostCheckCommand,
		HostZone:                 cli.HostZone,
		StaticHostVars:           cli.StaticHostVars,
		HostCacheTTL:             cli.HostCacheTTL,
		HostKeepFor:              cli.HostKeepFor,
	}

	conf.BearerTokens = nonEmpty(conf.BearerTokens)
	if len(conf.BearerTokens) == 0 {
		return &conf, errNoBearerToken
	}

	if conf.HostMappingFile != "" && len(conf.HostLabels) == 0 {
		return &conf, errMappingWithoutLabels
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

var (
	errNoBearerToken        = errors.New("at least one non-empty bearer token is required")
	errMappingWithoutLabels = errors.New("a host mapping file requires --host-labels")
)

// nonEmpty returns the trimmed, non-empty entries of the given list
func nonEmpty(list []string) []string {
	result := make([]string, 0, len(list))

	for _, s := range list {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}

	return result
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
