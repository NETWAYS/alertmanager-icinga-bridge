// Licensed under "BSD 3-Clause". See LICENSE file.

package gc

import (
	"fmt"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// extractDowntime searches the provided downtime array for a downtime for
// service with name svcName.
func extractDowntime(downtimes []icinga2.Downtime, svcName string) (icinga2.Downtime, bool) {
	for _, dt := range downtimes {
		if dt.Service == svcName {
			return dt, true
		}
	}
	return icinga2.Downtime{}, false
}

// collectService cleans up a single service that is managed by this alertmanager-icinga-bridge
func collectService(svc icinga2.Service, c config.Configuration, downtimes []icinga2.Downtime) error {
	logger := c.GetLogger()
	icinga := c.GetIcingaClient()

	_, heartbeat := svc.Vars["label_heartbeat"]
	_, downtimed := extractDowntime(downtimes, svc.Name)
	if heartbeat && !downtimed {
		logger.Info(fmt.Sprintf("[Collect] Skipping heartbeat %v: not downtimed", svc.Name))
		return nil
	} else if svc.State > 0 && !heartbeat {
		logger.Info(fmt.Sprintf("[Collect] Skipping service %v: state=%v, downtimed=%v",
			svc.Name, svc.State, downtimed))
		return nil
	}

	keepForNs := int64(svc.Vars["keep_for"].(float64))
	keepFor := time.Duration(keepForNs)
	lastChangeUnixNs := int64(svc.LastStateChange * 1e9)
	lastChange := time.Unix(0, lastChangeUnixNs)
	serviceAge := time.Since(lastChange)
	if serviceAge >= keepFor {
		logger.Info(fmt.Sprintf("[Collect] Deleting service %v: keep_for = %v; age = %v", svc.Name, keepFor, serviceAge))
		err := icinga.DeleteService(svc.FullName())
		if err != nil {
			logger.Error(fmt.Sprintf("Error while deleting service: %v", err))
		}
	} else {
		logger.Info(fmt.Sprintf("[Collect] Skipping service %v: keep_for = %v; age = %v", svc.Name, keepFor, serviceAge))
	}
	return nil
}

// Collect runs a garbage collection cycle to clean up any old
// alertmanager-icinga-bridge-managed service objects
func Collect(ts time.Time, c config.Configuration) error {
	logger := c.GetLogger()
	logger.Info(fmt.Sprintf("[Collect] Running garbage collection at ts=%v", ts))
	// Get all alertmanager-icinga-bridge services
	icinga := c.GetIcingaClient()
	hostname := c.GetConfig().HostName
	services, err := icinga.ListServices(icinga2.QueryFilter{
		Filter: fmt.Sprintf(`match("%v", service.host_name)`, hostname),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("[Collect] Error while listing services: %v", err))
		return err
	}
	logger.Info(fmt.Sprintf("[Collect] Found %v services with host = %v", len(services), hostname))
	downtimes, err := icinga.ListDowntimes(icinga2.QueryFilter{
		Filter: fmt.Sprintf(`match("%v", downtime.host_name)`, hostname),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("[Collect] Error while listing downtimes: %v", err))
		return err
	}
	logger.Info(fmt.Sprintf("[Collect] Found %v downtimes with host = %v", len(downtimes), hostname))
	// Iterate through services, finding ones that are managed by this
	// alertmanager-icinga-bridge and delete services which have transitioned to OK longer
	// than keep_for ago
	for _, svc := range services {
		if svc.Vars["bridge_uuid"] == c.GetConfig().UUID {
			logger.Info(fmt.Sprintf("[Collect] Found service %v with our bridge UUID", svc.Name))
			err = collectService(svc, c, downtimes)
			if err != nil {
				logger.Error(fmt.Sprintf("[Collect] Error garbage-collecting service: %v", err))
			}
		}
	}
	logger.Info(fmt.Sprintf("[Collect] Garbage collection completed in %v", time.Since(ts)))
	return nil
}
