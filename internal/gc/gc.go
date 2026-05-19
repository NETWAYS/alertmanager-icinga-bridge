// SPDX-License-Identifier: BSD-3-Clause

package gc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// GarbageCollector handles regular synchronization with Icinga
type GarbageCollector struct {
	mutex        sync.Mutex
	running      bool
	serviceQuery icinga2.QueryFilter
	serviceName  string
	config       *config.Config
	logger       *slog.Logger
	icingaClient *icinga2.Client
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
		serviceName: fmt.Sprintf("%s!%s", config.IcingaHostname, config.HeartbeatService),
	}

	return s
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

				g.logger.Debug("Removing services from Icinga", "component", "gc")
				// Get all services for the bridge host
				ctxIcinga, cancelIcinga := context.WithTimeout(ctx, 20*time.Second)
				defer cancelIcinga()

				services, errSvc := g.icingaClient.GetServices(ctxIcinga, g.serviceQuery)

				if errSvc != nil {
					g.logger.Error("Could not fetch services for removal from Icinga", "component", "gc", "error", errSvc.Error())
					return
				}

				if len(services) == 0 {
					g.logger.Debug("No services to remove from Icinga", "component", "gc")
					return
				}

				for _, svc := range services {
					errSvcRemove := g.removeServiceIfRequired(ctxIcinga, svc)

					if errSvcRemove != nil {
						g.logger.Error("Could not remove service from Icinga", "component", "gc", "service", svc.Name, "error", errSvcRemove.Error())
						return
					}
				}

				g.logger.Debug("Removed services from Icinga", "component", "gc")
			}()
		}
	}
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
			defer cancelIcinga()

			// Get the heartbeat service for the bridge host
			svc, errSvc := g.icingaClient.GetService(ctxIcinga, g.serviceName)

			if errSvc != nil {
				g.logger.Error("Could not fetch heartbeat service from Icinga", "component", "gc", "error", errSvc.Error())
				return
			}

			action := icinga2.Action{
				ExitStatus:   0,
				PluginOutput: fmt.Sprintf("[OK] Last Alertmanager Bridge heartbeat at: %v", time.Now().Format(time.RFC3339)),
			}

			errProcess := g.icingaClient.ProcessCheckResult(ctxIcinga, svc, action)

			if errProcess != nil {
				g.logger.Error("Could not process-check-result for heartbeat", "component", "gc", "error", errProcess.Error())
				return
			}

			g.logger.Info("Successfully sent heartbeat to Icinga", "component", "gc", "service", g.serviceName)
			g.logger.Debug("Sent heartbeat to Icinga", "component", "gc")
		}
	}
}

// removeServiceIfRequired the given service if applicable
func (g *GarbageCollector) removeServiceIfRequired(ctx context.Context, service icinga2.Service) error {
	_, heartbeat := service.Vars["label_heartbeat"]

	if heartbeat && service.HasDowntime() {
		g.logger.Debug("Skipping heartbeat and not downtimed service", "component", "gc", "service", service.Name)
		return nil
	}

	if service.State > 0 && !heartbeat {
		g.logger.Debug("Skipping not-OK services that are not heartbeats", "component", "gc", "service", service.Name)
		return nil
	}

	if isSkipableDueToAge(service, g.config.KeepFor) {
		g.logger.Debug("Skipping service with smaller age than keep_for", "component", "gc", "service", service.Name)
		return nil
	}

	// Delete service
	svcName := service.FullName()
	errDel := g.icingaClient.DeleteService(ctx, svcName)

	if errDel != nil {
		g.logger.Error("Could not remove service", "component", "gc", "service", svcName, "error", errDel.Error())
		return fmt.Errorf("could remove service: %w", errDel)
	}

	g.logger.Info("Successfully removed service from Icinga", "component", "gc", "service", svcName)

	return nil
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
