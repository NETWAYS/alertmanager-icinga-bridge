// SPDX-License-Identifier: BSD-3-Clause

package hosts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// Host variables set on hosts created by the bridge.
const (
	// VarBridgeUUID holds the ID of the bridge instance that created the host.
	VarBridgeUUID = "bridge_uuid"
	// VarBridgeCreated holds the creation time (Unix seconds).
	VarBridgeCreated = "bridge_created"
	// SourceVarSuffix is appended to a label name for the variable that holds
	// the original value of a mapped label, e.g. vars.project_source.
	SourceVarSuffix = "_source"
)

// HostClient is the part of the Icinga client the Manager needs.
type HostClient interface {
	GetHost(ctx context.Context, name string) (icinga2.Host, error)
	CreateHost(ctx context.Context, name string, templates []string, attrs map[string]any) error
}

// ManagerConfig configures how hosts are created.
type ManagerConfig struct {
	// ID of the bridge instance, stored in vars.bridge_uuid.
	ID           string
	Templates    []string
	CheckCommand string
	Zone         string
	StaticVars   map[string]string
	// CacheTTL is how long a host is assumed to exist after it was seen.
	CacheTTL time.Duration
}

type call struct {
	done chan struct{}
	err  error
}

// Manager makes sure hosts exist in Icinga, creating them if necessary.
// Known hosts are cached so that most alerts cost no extra API call.
type Manager struct {
	cfg    ManagerConfig
	client HostClient
	logger *slog.Logger
	now    func() time.Time

	mu       sync.Mutex
	known    map[string]time.Time
	inflight map[string]*call
}

// NewManager returns a new Manager.
func NewManager(cfg ManagerConfig, client HostClient, logger *slog.Logger) *Manager {
	return &Manager{
		cfg:      cfg,
		client:   client,
		logger:   logger,
		now:      time.Now,
		known:    map[string]time.Time{},
		inflight: map[string]*call{},
	}
}

// Ensure makes sure the target host exists. Concurrent calls for the same host
// share one lookup/creation.
func (m *Manager) Ensure(ctx context.Context, t Target) error {
	m.mu.Lock()

	if seen, ok := m.known[t.Name]; ok && m.now().Sub(seen) < m.cfg.CacheTTL {
		m.mu.Unlock()
		return nil
	}

	if c, ok := m.inflight[t.Name]; ok {
		m.mu.Unlock()

		select {
		case <-c.done:
			return c.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	c := &call{done: make(chan struct{})}
	m.inflight[t.Name] = c
	m.mu.Unlock()

	c.err = m.ensure(ctx, t)

	m.mu.Lock()
	delete(m.inflight, t.Name)

	if c.err == nil {
		m.known[t.Name] = m.now()
	}
	m.mu.Unlock()

	close(c.done)

	return c.err
}

// Forget drops a host from the cache, e.g. before it is deleted or after
// Icinga reported it missing.
func (m *Manager) Forget(name string) {
	m.mu.Lock()
	delete(m.known, name)
	m.mu.Unlock()
}

// Attrs returns the attributes a new host is created with.
func (m *Manager) Attrs(t Target) map[string]any {
	vars := make(map[string]any, len(m.cfg.StaticVars)+2*len(t.Labels)+2)

	for k, v := range m.cfg.StaticVars {
		vars[k] = v
	}

	for label, value := range t.Labels {
		vars[label] = value
	}

	for label, raw := range t.Sources {
		vars[label+SourceVarSuffix] = raw
	}

	vars[VarBridgeUUID] = m.cfg.ID
	vars[VarBridgeCreated] = m.now().Unix()

	attrs := map[string]any{
		"display_name": t.Name,
		"vars":         vars,
	}

	if m.cfg.CheckCommand != "" {
		attrs["check_command"] = m.cfg.CheckCommand
	}

	if m.cfg.Zone != "" {
		attrs["zone"] = m.cfg.Zone
	}

	return attrs
}

func (m *Manager) ensure(ctx context.Context, t Target) error {
	_, err := m.client.GetHost(ctx, t.Name)
	if err == nil {
		return nil
	}

	if !errors.Is(err, icinga2.ErrNotFound) {
		return fmt.Errorf("could not look up host %s: %w", t.Name, err)
	}

	err = m.client.CreateHost(ctx, t.Name, m.cfg.Templates, m.Attrs(t))
	if errors.Is(err, icinga2.ErrAlreadyExists) {
		m.logger.Debug("Host was created concurrently", "component", "hosts", "host", t.Name)
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not create host %s: %w", t.Name, err)
	}

	m.logger.Info("Created host for incoming alerts", "component", "hosts", "host", t.Name, "labels", t.Labels)

	return nil
}

// CreatedAt returns the creation time stored on a host by the bridge.
func CreatedAt(vars icinga2.Vars) (time.Time, bool) {
	switch v := vars[VarBridgeCreated].(type) {
	case float64:
		return time.Unix(int64(v), 0), true
	case int64:
		return time.Unix(v, 0), true
	case int:
		return time.Unix(int64(v), 0), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return time.Unix(i, 0), true
		}
	}

	return time.Time{}, false
}
