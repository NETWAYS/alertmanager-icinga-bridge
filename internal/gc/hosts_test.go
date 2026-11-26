// SPDX-License-Identifier: BSD-3-Clause

package gc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

type forgetter struct {
	mu        sync.Mutex
	forgotten []string
}

func (f *forgetter) Forget(name string) {
	f.mu.Lock()
	f.forgotten = append(f.forgotten, name)
	f.mu.Unlock()
}

func TestCollectHosts(t *testing.T) {
	now := time.Unix(1700000000, 0)
	old := float64(now.Add(-200 * time.Hour).Unix())
	recent := float64(now.Add(-time.Hour).Unix())

	hostList := []map[string]any{
		{"name": "empty-old", "vars": map[string]any{"bridge_uuid": "unittest", "bridge_created": old}},
		{"name": "empty-recent", "vars": map[string]any{"bridge_uuid": "unittest", "bridge_created": recent}},
		{"name": "busy-old", "vars": map[string]any{"bridge_uuid": "unittest", "bridge_created": old}},
		{"name": "new-service", "vars": map[string]any{"bridge_uuid": "unittest", "bridge_created": old}},
		{"name": "foreign", "vars": map[string]any{"bridge_uuid": "other", "bridge_created": old}},
		{"name": "no-created", "vars": map[string]any{"bridge_uuid": "unittest"}},
		{"name": "unittest", "vars": map[string]any{"bridge_uuid": "unittest", "bridge_created": old}},
	}

	var (
		mu      sync.Mutex
		deleted []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := url.PathUnescape(r.URL.EscapedPath())

		var filter icinga2.QueryFilter

		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &filter)

		results := []any{}

		switch {
		case r.Method == http.MethodGet && path == "/v1/objects/hosts/":
			if filter.Filter != `host.vars.bridge_uuid == "unittest"` {
				t.Errorf("unexpected host filter %q", filter.Filter)
			}

			for _, h := range hostList {
				results = append(results, map[string]any{"attrs": h})
			}
		case r.Method == http.MethodGet && path == "/v1/objects/services/":
			switch {
			case filter.Filter == `service.vars.bridge_uuid == "unittest"`:
				// A firing service keeps its host
				results = append(results, map[string]any{"attrs": map[string]any{"name": "svc", "host_name": "busy-old", "state": 2}})
			case strings.Contains(filter.Filter, `host.name == "new-service"`):
				results = append(results, map[string]any{"attrs": map[string]any{"name": "svc", "host_name": "new-service"}})
			}
		case r.Method == http.MethodDelete && strings.HasPrefix(path, "/v1/objects/hosts/"):
			if r.URL.Query().Get("cascade") != "1" {
				t.Errorf("host delete without cascade")
			}

			mu.Lock()
			deleted = append(deleted, strings.TrimPrefix(path, "/v1/objects/hosts/"))
			mu.Unlock()

			return
		default:
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	defer ts.Close()

	config := testConfig(ts.URL)
	config.ID = "unittest"
	config.HostKeepFor = 168 * time.Hour

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cache := &forgetter{}
	g := NewGarbageCollector(config, logger, icinga2.NewClient(config, logger)).WithHostCache(cache)
	g.now = func() time.Time { return now }

	g.collect(context.Background())

	if !slices.Equal(deleted, []string{"empty-old"}) {
		t.Fatalf("want only empty-old deleted, got %v", deleted)
	}

	if !slices.Contains(cache.forgotten, "empty-old") {
		t.Errorf("deleted host must be dropped from the cache, got %v", cache.forgotten)
	}

	// Disabled host removal
	deleted = nil
	config.HostKeepFor = 0
	g.collect(context.Background())

	if len(deleted) != 0 {
		t.Fatalf("host removal is disabled, but deleted %v", deleted)
	}
}
