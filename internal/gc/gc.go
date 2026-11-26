// SPDX-License-Identifier: BSD-3-Clause

// Package gc provides the garbage collector that handles cleanup at the Icinga API
package gc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/hosts"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// GarbageCollector handles regular synchronization with Icinga
type GarbageCollector struct {
	mutex        sync.Mutex
	running      bool
	serviceQuery icinga2.QueryFilter
	hostQuery    icinga2.QueryFilter
	serviceName  string
	hostCache    HostForgetter
	now          func() time.Time
	config       *config.Config
	logger       *slog.Logger
	icingaClient *icinga2.Client
}

// HostForgetter drops hosts from the cache of known hosts
type HostForgetter interface {
	Forget(name string)
}

// NewGarbageCollector returns a new GarbageCollector based on the configuration
func NewGarbageCollector(config *config.Config, logger *slog.Logger, icingaClient *icinga2.Client) *GarbageCollector {
	s := &GarbageCollector{
		config:       config,
		logger:       logger,
		icingaClient: icingaClient,
		serviceQuery: icinga2.QueryFilter{
			Filter: fmt.Sprintf(`service.vars.bridge_uuid == "%s"`, config.ID),
		},
		hostQuery: icinga2.QueryFilter{
			Filter: fmt.Sprintf(`host.vars.%s == %q`, hosts.VarBridgeUUID, config.ID),
		},
		serviceName: fmt.Sprintf("%s!%s", config.IcingaHostname, config.HeartbeatService),
		now:         time.Now,
	}

	return s
}

// WithHostCache lets the garbage collector drop removed hosts from the cache
func (g *GarbageCollector) WithHostCache(cache HostForgetter) *GarbageCollector {
	g.hostCache = cache

	return g
}

// Start starts this Syncher
func (g *GarbageCollector) Start(ctx context.Context) {
	go func() {
		g.logger.Debug("Starting garbage collection", "component", "gc")
		g.start(ctx)
	}()
	go func() {
		g.logger.Debug("Starting heartbeat ticker", "component", "gc")
		g.heartbeat(ctx)
	}()
}

func (g *GarbageCollector) start(ctx context.Context) {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	ticker := time.NewTicker(g.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("GarbageCollector received shutdown", "component", "gc")
			return
		case <-ticker.C:
			g.mutex.Lock()

			if g.running {
				g.mutex.Unlock()
				g.logger.Warn("Previous sync still running", "component", "gc")

				continue
			}

			g.running = true
			g.mutex.Unlock()

			go func() {
				defer func() {
					g.mutex.Lock()
					g.running = false
					g.mutex.Unlock()
				}()

				ctxIcinga, cancelIcinga := context.WithTimeout(ctx, gcTimeout)
				defer cancelIcinga()

				g.collect(ctxIcinga)
			}()
		}
	}
}

// Timeout for one garbage collection run
const gcTimeout = 5 * time.Minute

// collect removes services and then hosts that are no longer needed
func (g *GarbageCollector) collect(ctx context.Context) {
	g.logger.Debug("Removing services from Icinga", "component", "gc")

	services, errSvc := g.icingaClient.GetServices(ctx, g.serviceQuery)

	if errSvc != nil {
		g.logger.Error("Could not fetch services for removal from Icinga", "component", "gc", "error", errSvc.Error())
		return
	}

	// Hosts that still have services of this bridge
	inUse := make(map[string]struct{}, len(services))

	for _, svc := range services {
		removed, errSvcRemove := g.removeServiceIfRequired(ctx, svc)

		if errSvcRemove != nil {
			g.logger.Error("Could not remove service from Icinga", "component", "gc", "service", svc.Name, "error", errSvcRemove.Error())
		}

		if !removed {
			inUse[svc.HostName] = struct{}{}
		}
	}

	g.logger.Debug("Removed services from Icinga", "component", "gc")

	if g.config.HostKeepFor > 0 {
		g.collectHosts(ctx, inUse)
	}
}

// collectHosts removes hosts created by this bridge that have no services of this
// bridge left and were created more than --host-keep-for ago
func (g *GarbageCollector) collectHosts(ctx context.Context, inUse map[string]struct{}) {
	g.logger.Debug("Removing hosts from Icinga", "component", "gc")

	candidates, errHosts := g.icingaClient.GetHosts(ctx, g.hostQuery)

	if errHosts != nil {
		g.logger.Error("Could not fetch hosts for removal from Icinga", "component", "gc", "error", errHosts.Error())
		return
	}

	for _, host := range candidates {
		if _, ok := inUse[host.Name]; ok || !g.hostIsRemovable(host) {
			continue
		}

		errRemove := g.removeHost(ctx, host.Name)
		if errRemove != nil {
			g.logger.Error("Could not remove host from Icinga", "component", "gc", "host", host.Name, "error", errRemove.Error())
		}
	}

	g.logger.Debug("Removed hosts from Icinga", "component", "gc")
}

// hostIsRemovable checks the age and ownership of a host
func (g *GarbageCollector) hostIsRemovable(host icinga2.Host) bool {
	if host.Name == "" || host.Name == g.config.IcingaHostname {
		return false
	}

	if owner, _ := host.Vars[hosts.VarBridgeUUID].(string); owner != g.config.ID {
		return false
	}

	created, ok := hosts.CreatedAt(host.Vars)
	if !ok {
		g.logger.Debug("Skipping host without creation time", "component", "gc", "host", host.Name)
		return false
	}

	if g.now().Sub(created) <= g.config.HostKeepFor {
		g.logger.Debug("Skipping host with smaller age than host_keep_for", "component", "gc", "host", host.Name)
		return false
	}

	return true
}

// removeHost deletes a host unless an alert has created a service on it in the meantime
func (g *GarbageCollector) removeHost(ctx context.Context, name string) error {
	if g.hostCache != nil {
		g.hostCache.Forget(name)
	}

	services, errSvc := g.icingaClient.GetServices(ctx, icinga2.QueryFilter{
		Filter: fmt.Sprintf(`host.name == %q && service.vars.bridge_uuid == %q`, name, g.config.ID),
	})

	if errSvc != nil && !errors.Is(errSvc, icinga2.ErrNotFound) {
		return fmt.Errorf("could not check services of host: %w", errSvc)
	}

	if len(services) > 0 {
		g.logger.Debug("Skipping host that has new services", "component", "gc", "host", name)
		return nil
	}

	errDel := g.icingaClient.DeleteHost(ctx, name)
	if errDel != nil && !errors.Is(errDel, icinga2.ErrNotFound) {
		return fmt.Errorf("could not remove host: %w", errDel)
	}

	g.logger.Info("Successfully removed host from Icinga", "component", "gc", "host", name)

	return nil
}

func (g *GarbageCollector) heartbeat(ctx context.Context) {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	ticker := time.NewTicker(g.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Heartbeat received shutdown", "component", "gc")
			return
		case <-ticker.C:
			g.logger.Debug("Sending heartbeat to Icinga", "component", "gc")

			ctxIcinga, cancelIcinga := context.WithTimeout(ctx, 20*time.Second)
			errBeat := g.sendHeartbeat(ctxIcinga)

			cancelIcinga()

			if errBeat != nil {
				// Skip this tick, we will try next time
				continue
			}

			g.logger.Info("Successfully sent heartbeat to Icinga", "component", "gc", "service", g.serviceName)
			g.logger.Debug("Sent heartbeat to Icinga", "component", "gc")
		}
	}
}

// sendHeartbeat submits the heartbeat result
func (g *GarbageCollector) sendHeartbeat(ctx context.Context) error {
	// Get the heartbeat service for the bridge host
	svc, errSvc := g.icingaClient.GetService(ctx, g.serviceName)

	if errSvc != nil {
		g.logger.Error("Could not fetch heartbeat service from Icinga", "component", "gc", "error", errSvc.Error())
		return errSvc
	}

	action := icinga2.Action{
		ExitStatus:   0,
		Filter:       fmt.Sprintf("host.name==%q && service.name==%q", svc.HostName, svc.Name),
		Type:         "Service",
		PluginOutput: fmt.Sprintf("[OK] Last Alertmanager Bridge heartbeat at: %v", time.Now().Format(time.RFC3339)),
	}

	errProcess := g.icingaClient.ProcessCheckResult(ctx, action)

	if errProcess != nil {
		g.logger.Error("Could not process-check-result for heartbeat", "component", "gc", "error", errProcess.Error())
		return errProcess
	}

	return nil
}

// removeServiceIfRequired removes the given service if applicable and reports whether it was removed
func (g *GarbageCollector) removeServiceIfRequired(ctx context.Context, service icinga2.Service) (bool, error) {
	_, heartbeat := service.Vars["label_heartbeat"]

	if heartbeat && !service.HasDowntime() {
		g.logger.Debug("Skipping heartbeat and not downtimed service", "component", "gc", "service", service.Name)
		return false, nil
	}

	if service.State > 0 && !heartbeat {
		g.logger.Debug("Skipping not-OK services that are not heartbeats", "component", "gc", "service", service.Name)
		return false, nil
	}

	if isSkipableDueToAge(service, g.config.KeepFor) {
		g.logger.Debug("Skipping service with smaller age than keep_for", "component", "gc", "service", service.Name)
		return false, nil
	}

	// Delete service
	svcName := service.FullName()
	errDel := g.icingaClient.DeleteService(ctx, svcName)

	if errDel != nil {
		g.logger.Error("Could not remove service", "component", "gc", "service", svcName, "error", errDel.Error())
		return false, fmt.Errorf("could not remove service: %w", errDel)
	}

	g.logger.Info("Successfully removed service from Icinga", "component", "gc", "service", svcName)

	return true, nil
}

// isSkipableDueToAge figures out if we can skip this service
func isSkipableDueToAge(service icinga2.Service, defaultKeepFor time.Duration) bool {
	keepForValue, ok := service.Vars["keep_for"]

	if !ok {
		return false
	}

	var keepFor time.Duration

	switch val := keepForValue.(type) {
	case float64:
		keepFor = time.Duration(int64(val))
	default:
		keepFor = defaultKeepFor
	}

	lastChangeTimestampInNs := int64(service.LastStateChange * 1e9)
	serviceAge := time.Since(time.Unix(0, lastChangeTimestampInNs))

	//nolint: staticcheck
	if serviceAge <= keepFor {
		return true
	}

	return false
}
