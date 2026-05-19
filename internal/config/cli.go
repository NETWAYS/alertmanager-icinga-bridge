// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	// General
	ID       string           `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_ID',help='Instance ID'"`
	Loglevel string           `kong:"default='info',env='ALERTMANAGER_ICINGA_BRIDGE_LOGLEVEL',help='Loglevel (debug, info, warn, error)'"`
	Version  kong.VersionFlag `kong:"help='Print version information and quit'"`

	// Icinga Client
	IcingaURL                []string          `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_URL',help='Icinga API URL (can be repeated)'"`
	IcingaHostname           string            `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_HOSTNAME',help='Icinga host name to manage services for'"`
	DisableKeepAlives        bool              `kong:"default=false,env='ALERTMANAGER_ICINGA_BRIDGE_DISABLE_KEEPALIVES',help='Disable HTTP keepalives'"`
	DisplayNameAsServiceName bool              `kong:"default='false',env='ALERTMANAGER_ICINGA_BRIDGE_DISPLAY_NAME_AS_SERVICE_NAME',help='Set the Icinga service display name to the generated service name'"`
	IcingaInsecureTLS        bool              `kong:"default=false,env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_INSECURE_TLS',help='Skip Icinga TLS verification'"`
	IcingaCAFile             string            `kong:"env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_CA',help='Path of a custom CA certificate to use when connecting to the Icinga API'"`
	IcingaPassword           string            `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_PASSWORD',help='Icinga API password'"`
	IcingaUser               string            `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_ICINGA_USERNAME',help='Icinga API username'"`
	CustomSeverityLevels     map[string]string `kong:"env='ALERTMANAGER_ICINGA_BRIDGE_ALERTMANAGER_CUSTOM_SEVERITY_LEVELS',help='Add or override the default mapping of severity levels to service states (severity_level=service_state)'"`

	// Garbage Collector
	GCInterval        time.Duration `kong:"default='15m',env='ALERTMANAGER_ICINGA_BRIDGE_GC_INTERVAL',help='Interval to check for and remove created services'"`
	HeartbeatInterval time.Duration `kong:"default='1m',env='ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_INTERVAL',help='Interval for the bridge self-monitoring service heartbeat'"`
	HeartbeatService  string        `kong:"default='heartbeat',env='ALERTMANAGER_ICINGA_BRIDGE_HEARTBEAT_SERVICE',help='The name for the bridge self-monitoring service'"`

	// Alertmanager Webhook Receiver
	ListenAddr              string            `kong:"default='127.0.0.1:8888',env='ALERTMANAGER_ICINGA_BRIDGE_LISTEN_ADDR',help='Listening address for the incoming Alertmanager requests'"`
	BearerToken             string            `kong:"required,env='ALERTMANAGER_ICINGA_BRIDGE_BEARER_TOKEN',help='Bearer token for incoming requests'"`
	TLSCertPath             string            `kong:"env='ALERTMANAGER_ICINGA_BRIDGE_TLS_CERT',help='Path of a certificate file for TLS-enabled webhook endpoint (full chain)'"`
	TLSKeyPath              string            `kong:"env='ALERTMANAGER_ICINGA_BRIDGE_TLS_KEY',help='Path of a private key file for TLS-enabled webhook endpoint'"`
	CheckCommand            string            `kong:"default='dummy',env='ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_COMMAND',help='Specify Icinga check command during service creation'"`
	ActiveChecks            bool              `kong:"default=false,env='ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_ACTIVE',help='Create Icinga services as active checks'"`
	PluginOutputByStates    bool              `kong:"default=false,env='ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_BY_STATES',help='Enable dynamic selection of plugin output annotation based on service state'"`
	MaxCheckAttempts        int               `kong:"default=1,env='ALERTMANAGER_ICINGA_BRIDGE_SERVICE_MAX_CHECK_ATTEMPTS',help='The maximum number of checks which are executed before changing to a hard state'"`
	Templates               []string          `kong:"default='generic-service',env='ALERTMANAGER_ICINGA_BRIDGE_SERVICE_TEMPLATE',help='Create Icinga services with the given template (can be repeated)'"`
	PluginOutputAnnotations []string          `kong:"default='message',env='ALERTMANAGER_ICINGA_BRIDGE_PLUGINOUTPUT_ANNOTATIONS',help='List of Annotation names to be used to set the plugin output for the Icinga service'"`
	ChecksInterval          time.Duration     `kong:"default='12h',env='ALERTMANAGER_ICINGA_BRIDGE_SERVICE_CHECKS_INTERVAL',help='Interval (in seconds) to be used for Icinga check_interval and retry_interval'"`
	KeepFor                 time.Duration     `kong:"default='168h',env='ALERTMANAGER_ICINGA_BRIDGE_KEEP_FOR',help='How long to keep created alerts around after they have been resolved'"`
	StaticServiceVars       map[string]string `kong:"env='ALERTMANAGER_ICINGA_BRIDGE_STATIC_SERVICE_VAR',help='Custom variable to be set for created Icinga services (variable=value, can be repeated)'"`
}
