// SPDX-License-Identifier: BSD-3-Clause

package hosts

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

type fakeClient struct {
	mu        sync.Mutex
	hosts     map[string]bool
	gets      atomic.Int32
	creates   atomic.Int32
	createErr error
	attrs     map[string]any
	templates []string
	delay     time.Duration
}

func (f *fakeClient) GetHost(_ context.Context, name string) (icinga2.Host, error) {
	f.gets.Add(1)
	time.Sleep(f.delay)

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.hosts[name] {
		return icinga2.Host{Name: name}, nil
	}

	return icinga2.Host{}, icinga2.ErrNotFound
}

func (f *fakeClient) CreateHost(_ context.Context, name string, templates []string, attrs map[string]any) error {
	f.creates.Add(1)

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createErr != nil {
		return f.createErr
	}

	f.hosts[name] = true
	f.attrs = attrs
	f.templates = templates

	return nil
}

func newTestManager(client *fakeClient) *Manager {
	m := NewManager(ManagerConfig{
		ID:           "unittest",
		Templates:    []string{"generic-host"},
		CheckCommand: "dummy",
		Zone:         "master",
		StaticVars:   map[string]string{"source": "alertmanager"},
		CacheTTL:     time.Minute,
	}, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.now = func() time.Time { return time.Unix(1700000000, 0) }

	return m
}

func TestEnsure_CreatesHostOnce(t *testing.T) {
	client := &fakeClient{hosts: map[string]bool{}, delay: 10 * time.Millisecond}
	m := newTestManager(client)

	target := Target{
		Name:    "xraydmz2-system",
		Labels:  map[string]string{"cluster": "xraydmz2", "project": "system"},
		Sources: map[string]string{"project": "p-q2v7w"},
	}

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := m.Ensure(context.Background(), target); err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	// Cached now
	if err := m.Ensure(context.Background(), target); err != nil {
		t.Fatal(err)
	}

	if got := client.creates.Load(); got != 1 {
		t.Fatalf("want 1 create, got %d", got)
	}

	if got := client.gets.Load(); got != 1 {
		t.Fatalf("want 1 lookup, got %d", got)
	}

	vars, _ := client.attrs["vars"].(map[string]any)

	want := map[string]any{
		"cluster":        "xraydmz2",
		"project":        "system",
		"project_source": "p-q2v7w",
		"source":         "alertmanager",
		VarBridgeUUID:    "unittest",
		VarBridgeCreated: int64(1700000000),
	}

	for k, v := range want {
		if vars[k] != v {
			t.Errorf("vars.%s: want %v, got %v", k, v, vars[k])
		}
	}

	if client.attrs["check_command"] != "dummy" || client.attrs["zone"] != "master" || client.attrs["display_name"] != "xraydmz2-system" {
		t.Errorf("unexpected attrs %v", client.attrs)
	}

	if len(client.templates) != 1 || client.templates[0] != "generic-host" {
		t.Errorf("unexpected templates %v", client.templates)
	}

	// Forget makes the next call look the host up again (without creating it)
	m.Forget(target.Name)

	if err := m.Ensure(context.Background(), target); err != nil {
		t.Fatal(err)
	}

	if client.gets.Load() != 2 || client.creates.Load() != 1 {
		t.Fatalf("want 2 lookups and 1 create, got %d and %d", client.gets.Load(), client.creates.Load())
	}
}

func TestEnsure_Errors(t *testing.T) {
	client := &fakeClient{hosts: map[string]bool{}, createErr: icinga2.ErrAlreadyExists}
	m := newTestManager(client)

	if err := m.Ensure(context.Background(), Target{Name: "a"}); err != nil {
		t.Fatalf("an existing host is fine, got %v", err)
	}

	boom := errors.New("boom")
	client.createErr = boom

	if err := m.Ensure(context.Background(), Target{Name: "b"}); !errors.Is(err, boom) {
		t.Fatalf("want boom, got %v", err)
	}

	// Failures are not cached
	client.createErr = nil

	if err := m.Ensure(context.Background(), Target{Name: "b"}); err != nil {
		t.Fatal(err)
	}
}

func TestCreatedAt(t *testing.T) {
	for _, v := range []any{float64(1700000000), int64(1700000000), "1700000000"} {
		got, ok := CreatedAt(icinga2.Vars{VarBridgeCreated: v})
		if !ok || got.Unix() != 1700000000 {
			t.Errorf("%T: got %v %v", v, got, ok)
		}
	}

	if _, ok := CreatedAt(icinga2.Vars{}); ok {
		t.Error("missing var must not parse")
	}
}
