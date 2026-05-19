// SPDX-License-Identifier: BSD-3-Clause

package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

var (
	errNotAMappingKey         = errors.New("key does meet the mappable pattern")
	errUnknownMappingType     = errors.New("unknown type")
	pluginOutputStateSuffixes = []string{"ok", "warning", "critical", "unknown"}
	mappingKeyPattern         = regexp.MustCompile("^icinga_([a-z]+)_(.*)$")
	serviceNamePattern        = regexp.MustCompile(`^[-+_.:,a-zA-Z0-9 %]{1,128}$`)
)

// Listener represents the daemon's API
type Listener struct {
	mux                  http.Handler
	config               *config.Config
	logger               *slog.Logger
	icingaClient         *icinga2.Client
	serviceNameValidator *regexp.Regexp
}

// NewListener returns a new Listener based on the given configuration
func NewListener(config *config.Config, logger *slog.Logger, icingaClient *icinga2.Client) *Listener {
	l := &Listener{
		config:               config,
		logger:               logger,
		icingaClient:         icingaClient,
		serviceNameValidator: serviceNamePattern,
	}

	mux := http.NewServeMux()
	// Register all handler functions here to have a central overview of the API
	mux.HandleFunc("GET /healthz", l.handleHealthy)
	mux.HandleFunc("POST /webhook", l.handleIncomingAlert)

	l.mux = authHandler(mux, config.BearerToken)

	return l
}

// Run starts the Listener
func (l *Listener) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:        l.config.ListenAddr,
		Handler:     l.mux,
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	// We start ListenAndServe in a goroutine here so that
	// the main routine can handle the signals and defer the shutdown properly.
	go func() {
		l.logger.Info("Started Listener", "component", "listener", "port", l.config.ListenAddr)

		var errServe error

		if l.config.TLSCertPath != "" && l.config.TLSKeyPath != "" {
			errServe = server.ListenAndServeTLS(l.config.TLSCertPath, l.config.TLSKeyPath)
		} else {
			errServe = server.ListenAndServe()
		}

		if !errors.Is(errServe, http.ErrServerClosed) {
			l.logger.Error("HTTP server error", "component", "listener", "error", errServe.Error())
		}

		l.logger.Info("Received Shutdown. Stopped serving new connections.", "component", "listener")
	}()

	// The signal channel will block until we registered signals are received.
	// We will then use a context with a timeout to shutdown the application gracefully.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctxShutdown, shutdownCancel := context.WithTimeout(ctx, 20*time.Second)
	defer shutdownCancel()

	// We are using Shutdown() with a timeout to gracefully
	// shut down the server without interrupting any active connections.
	errShut := server.Shutdown(ctxShutdown)

	if errShut != nil {
		l.logger.Error("HTTP shutdown error", "component", "listener", "error", errShut.Error())
		return errShut
	}

	l.logger.Info("Completed shutdown", "component", "listener")

	return nil
}

// authHandler is a Middleware for handling authorization.
// We currently only need Bearer token authorization, since we don't have complex actions in the API.
func authHandler(next http.Handler, expectedToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization Header from the request
		authHeader := r.Header.Get("Authorization")

		scheme, receivedToken, found := strings.Cut(authHeader, " ")

		if !found || !strings.EqualFold(scheme, "Bearer") || receivedToken == "" {
			http.Error(w, "malformed authorization header", http.StatusUnauthorized)
			return
		}

		if receivedToken != expectedToken {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Pass down the request to the next middleware or handler
		next.ServeHTTP(w, r)
	})
}

// handleHealthy handles health checks
func (l *Listener) handleHealthy(w http.ResponseWriter, _ *http.Request) {
	l.logger.Debug("Handling health check", "component", "listener")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, `{"status": "ok"}`)

	l.logger.Debug("Handled health check", "component", "listener")
}

// handleIncomingAlert handles the incoming alerts from the Alertmanager
func (l *Listener) handleIncomingAlert(w http.ResponseWriter, r *http.Request) {
	l.logger.Debug("Handling incoming alert", "component", "listener")

	var payload WebhookPayload

	errDecode := json.NewDecoder(r.Body).Decode(&payload)

	if errDecode != nil {
		l.logger.Error("Received invalid JSON", "component", "listener", "error", errDecode.Error())
		http.Error(w, "invalid JSON:"+errDecode.Error(), http.StatusBadRequest)

		return
	}

	errManage := l.manageIcingaService(r.Context(), payload)

	if errManage != nil {
		l.logger.Error("Could not manage Icinga service for incoming alert", "component", "listener", "error", errManage.Error())
		http.Error(w, "Could not manage Icinga service for incoming alert", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	// Transform and handle with icingaClient
	l.logger.Debug("Handled incoming alert", "component", "listener")
}

// manageIcingaService talks to the Icinga API to manage the service for the incoming alert
func (l *Listener) manageIcingaService(ctx context.Context, payload WebhookPayload) error {
	l.logger.Debug("Managing Icinga service", "component", "listener")

	ctxIcinga, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, errHost := l.icingaClient.GetHost(ctxIcinga, l.config.IcingaHostname)

	if errHost != nil {
		return errHost
	}

	for _, alert := range payload.Alerts {
		l.logger.Debug("Handling alert", "component", "listener", "alert", alert)

		displayName, ok := alert.Labels["alertname"]

		if !ok {
			l.logger.Warn("alert does not have label 'alertname'", "alert", alert)
		}

		serviceName := generateServiceName(l.config.ID, alert)

		if !l.serviceNameValidator.MatchString(serviceName) {
			return fmt.Errorf("service name '%v' does not match Icinga constraints", serviceName)
		}

		if l.config.DisplayNameAsServiceName {
			displayName = serviceName
		}

		exitCode := severityToExitCode(alert.Status, alert.Labels["severity"], l.config.MergedSeverityLevels)

		svc, errUpsert := l.updateOrCreateService(ctx, serviceName, displayName, exitCode, alert)

		if errUpsert != nil {
			return errUpsert
		}

		// If we got an empty service object, the service was not
		// created, don't try to call process-check-result
		if svc.Name == "" {
			l.logger.Warn("Got empty service object for alert: " + displayName)

			continue
		}

		// Get the Plugin Output from the first Annotation we find that has some data
		pluginOutput := l.generatePluginOutput(alert, exitCode)

		action := icinga2.Action{
			ExitStatus:   exitCode,
			PluginOutput: pluginOutput,
		}

		errProcess := l.icingaClient.ProcessCheckResult(ctx, svc, action)

		if errProcess != nil {
			return errProcess
		}
	}

	l.logger.Debug("Managed Icinga service", "component", "listener")

	return nil
}

func (l *Listener) updateOrCreateService(ctx context.Context, serviceName, displayName string, exitCode int, alert Alert) (icinga2.Service, error) {
	heartbeatInterval := time.Duration(0)

	if val, ok := alert.Labels["heartbeat"]; ok {
		if alert.Status == alertStatusResolved {
			// Not processing resolved heartbeats
			return icinga2.Service{}, nil
		}

		interval, errParse := time.ParseDuration(val)

		if errParse != nil {
			return icinga2.Service{}, fmt.Errorf("unable to parse heartbeat interval: %w", errParse)
		}

		heartbeatInterval = interval
	}

	svc := l.prepareService(serviceName, displayName, alert, exitCode, heartbeatInterval)
	svcFullname := svc.FullName()

	_, errGet := l.icingaClient.GetService(ctx, svcFullname)

	// There was an error, either no object or something went wrong
	if errGet != nil {
		if errors.Is(errGet, icinga2.ErrNotFound) {
			// No such object, we need to create one and then return
			l.logger.Debug("Creating new service for incoming alert", "component", "listener", "service", svcFullname)

			errCreate := l.icingaClient.CreateService(ctx, svc)

			if errCreate != nil {
				return svc, fmt.Errorf("unable to create service: %w", errCreate)
			}

			l.logger.Info("Successfully created service for incoming alert", "component", "listener", "service", svcFullname)

			return svc, nil
		}

		return svc, fmt.Errorf("update to fetch service status from Icinga: %w", errGet)
	}

	// We got a service in Icinga that we can update
	// Templates needs to be ignored if the service is already created due to the Error:
	// Attribute 'templates' could not be set: Error: Attribute cannot be modified.
	svc.Templates = nil

	l.logger.Debug("Updating existing service for incoming alert", "component", "listener", "service", svcFullname)

	errUpdate := l.icingaClient.UpdateService(ctx, svc)

	if errUpdate != nil {
		return svc, fmt.Errorf("unable to update existing service: %w", errUpdate)
	}

	l.logger.Info("Successfully updated service for incoming alert", "component", "listener", "service", svcFullname)

	return svc, nil
}

// prepareService creates a service from the alert and other configured data
func (l *Listener) prepareService(serviceName string, displayName string, alert Alert, status int, heartbeatInterval time.Duration) icinga2.Service {
	serviceVars := make(icinga2.Vars, 2+len(l.config.StaticServiceVars))

	serviceVars["bridge_uuid"] = l.config.ID
	serviceVars["keep_for"] = l.config.KeepFor
	serviceVars = mapIcingaVariables(serviceVars, alert.Labels, "label_")
	serviceVars = mapIcingaVariables(serviceVars, alert.Annotations, "annotation_")
	// addStaticIcingaVariables merged the given map into the existing vars and does not override existing key
	for k, v := range l.config.StaticServiceVars {
		if _, ok := serviceVars[k]; !ok {
			serviceVars[k] = v
		}
	}

	svc := icinga2.Service{
		Name:               serviceName,
		DisplayName:        displayName,
		HostName:           l.config.IcingaHostname,
		CheckCommand:       l.config.CheckCommand,
		EnableActiveChecks: l.config.ActiveChecks,
		Notes:              alert.Annotations["description"],
		ActionURL:          alert.GeneratorURL,
		NotesURL:           alert.Annotations["runbook_url"],
		CheckInterval:      l.config.ChecksInterval.Seconds(),
		RetryInterval:      l.config.ChecksInterval.Seconds(),
		// We don't usually need soft states in Icinga, since the grace
		// periods are already managed by Prometheus/Alertmanager and relevant
		// config parameter defaults to 1, but is still tunable for other usecases
		MaxCheckAttempts: l.config.MaxCheckAttempts,
		Templates:        l.config.IcingaTemplates,
		Vars:             serviceVars,
	}

	if value, ok := alert.Labels["icinga_use_host"]; ok {
		svc.HostName = value
	}

	if value, ok := alert.Labels["icinga_use_zone"]; ok {
		svc.Zone = value
	}

	if value, ok := alert.Labels["icinga_use_template"]; ok {
		svc.Templates = append(svc.Templates, value)
	}

	// Check if this is a heartbeat service and adjust serviceData accordingly
	if heartbeatInterval.Seconds() > 0.0 {
		// Set dummy text to message annotation on alert
		svc.Vars["dummy_text"] = alert.Annotations["message"]
		// Set exitStatus for missed heartbeat to Alert's severity
		svc.Vars["dummy_state"] = status
		// Add 10% onto requested check interval to allow some network latency for the check results
		svc.CheckInterval = heartbeatInterval.Seconds() * 1.1
		svc.RetryInterval = heartbeatInterval.Seconds() * 1.1
		// Enable active checks for heartbeat check
		svc.EnableActiveChecks = true
	}

	return svc
}

// generatePluginOutput generates the plugin output based on the alert
func (l *Listener) generatePluginOutput(alert Alert, exitCode int) string {
	for _, v := range l.config.PluginOutputAnnotations {
		// If the PluginOutputByStates option is enabled then first look for an annotation with the state suffix
		// otherwise fall back to just using the PluginOutputAnnotations value as is
		if l.config.PluginOutputByStates {
			// Note, I don't like PluginOutputStateSuffixes being a slide and exitCode being the index
			if value, ok := alert.Annotations[fmt.Sprintf("%s_%s", v, pluginOutputStateSuffixes[exitCode])]; ok {
				return value
			}
		}

		if value, ok := alert.Annotations[v]; ok {
			return value
		}
	}

	return ""
}

// generateServiceName generates a unique internal service name used for Icinga
// Uses the instance's OD to ensure we accidentally touch another instance's services
func generateServiceName(id string, alert Alert) string {
	hash := sha256.New()
	hash.Write([]byte(id))
	hash.Write([]byte(mapToStableString(alert.Labels)))
	labelhash := hex.EncodeToString(hash.Sum(nil))[:16] // 16 characters
	serviceName := alert.Labels["alertname"]
	serviceName = fmt.Sprintf("%v_%v", serviceName, labelhash)

	return serviceName
}

// mapToStableString converts a map of alert labels to a string
// representation which is stable if the same map of alert labels is provided
// to subsequent calls of mapToStableString.
func mapToStableString(data map[string]string) string {
	var keys []string

	for k := range data {
		if k != "severity" {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	var sb strings.Builder

	for _, k := range keys {
		s := fmt.Sprintf("%v:%v ", k, data[k])
		sb.WriteString(s)
	}

	return sb.String()
}

// severityToExitStatus computes an exit code which Icinga understands from
// an alert's status and severity label
func severityToExitCode(status string, severity string, severityLevels map[string]int) int {
	if status == alertStatusFiring {
		code, ok := severityLevels[strings.ToLower(severity)]

		if !ok {
			return 3
		}

		return code
	}

	if status == alertStatusResolved {
		return 0
	}

	return 3
}

func mapIcingaVariables(vars icinga2.Vars, labels map[string]string, prefix string) icinga2.Vars {
	for key, value := range labels {
		vars[prefix+key] = value

		subKey, subValue, err := mapIcingaVariable(key, value)

		if errors.Is(err, errNotAMappingKey) {
			continue
		}

		if err != nil {
			// Failed to map Icinga variable
			continue
		}

		vars[subKey] = subValue
	}

	return vars
}

func mapIcingaVariable(key, value string) (string, any, error) {
	matches := mappingKeyPattern.FindStringSubmatch(key)
	if len(matches) < 3 {
		return key, value, errNotAMappingKey
	}

	typ := matches[1]
	subKey := matches[2]

	switch typ {
	case "number":
		v, err := strconv.Atoi(value)

		if err != nil {
			return "", nil, err
		}

		return subKey, v, nil

	case "string":
		return subKey, value, nil
	}

	return "", nil, errUnknownMappingType
}
