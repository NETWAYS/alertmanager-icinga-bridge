// SPDX-License-Identifier: BSD-3-Clause

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/hosts"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// fakeIcinga is a minimal stateful stand-in for the Icinga API
type fakeIcinga struct {
	mu       sync.Mutex
	hosts    map[string]map[string]any
	services map[string]map[string]any
	results  []map[string]any
	requests []string
}

func newFakeIcinga() *fakeIcinga {
	return &fakeIcinga{hosts: map[string]map[string]any{}, services: map[string]map[string]any{}}
}

func (f *fakeIcinga) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	path, _ := url.PathUnescape(r.URL.EscapedPath())
	f.requests = append(f.requests, r.Method+" "+path)

	var body map[string]any

	data, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(data, &body)

	switch {
	case path == "/v1/actions/process-check-result/":
		f.results = append(f.results, body)
	case strings.HasPrefix(path, "/v1/objects/hosts/"):
		name := strings.TrimPrefix(path, "/v1/objects/hosts/")
		f.object(w, r, f.hosts, name, body)
	case strings.HasPrefix(path, "/v1/objects/services/"):
		name := strings.TrimPrefix(path, "/v1/objects/services/")
		host, _, _ := strings.Cut(name, "!")

		if _, ok := f.hosts[host]; !ok && r.Method == http.MethodPut {
			http.Error(w, `{"error": 500, "status": "Host does not exist"}`, http.StatusInternalServerError)
			return
		}

		f.object(w, r, f.services, name, body)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeIcinga) object(w http.ResponseWriter, r *http.Request, objects map[string]map[string]any, name string, body map[string]any) {
	switch r.Method {
	case http.MethodGet:
		attrs, ok := objects[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{map[string]any{"attrs": attrs}}})
	case http.MethodPut:
		if _, ok := objects[name]; ok {
			http.Error(w, `{"status": "Object already exists."}`, http.StatusInternalServerError)
			return
		}

		attrs, _ := body["attrs"].(map[string]any)
		attrs["name"] = name
		objects[name] = attrs
	case http.MethodPost:
		attrs, _ := body["attrs"].(map[string]any)
		for k, v := range attrs {
			objects[name][k] = v
		}
	}
}

func newHostListener(t *testing.T, ts *httptest.Server, mapping string) *Listener {
	t.Helper()

	cfg := testConfig(ts.URL)
	cfg.HostLabels = []string{"cluster", "project"}
	cfg.BearerTokens = []string{"t"}

	path := ""

	if mapping != "" {
		path = t.TempDir() + "/mapping.yaml"
		_ = writeFile(path, mapping)
	}

	resolver, err := hosts.NewResolver(cfg.HostLabels, "-", "", path)
	if err != nil {
		t.Fatal(err)
	}

	client := icinga2.NewClient(cfg, testLogger())
	manager := hosts.NewManager(hosts.ManagerConfig{
		ID:           cfg.ID,
		Templates:    []string{"generic-host"},
		CheckCommand: "dummy",
		CacheTTL:     time.Minute,
	}, client, testLogger())

	return NewListener(cfg, testLogger(), client).WithHosts(resolver, manager)
}

func clusterAlert(alertname, cluster, project string) Alert {
	labels := map[string]string{
		"alertname": alertname,
		"severity":  "critical",
		"cluster":   cluster,
	}

	if project != "" {
		labels["project"] = project
	}

	return Alert{
		Status:      "firing",
		Labels:      labels,
		Annotations: map[string]string{"message": alertname + " is firing"},
	}
}

func TestManageIcingaResults_DerivedHosts(t *testing.T) {
	icinga := newFakeIcinga()
	icinga.hosts["unittest"] = map[string]any{"name": "unittest"}

	ts := httptest.NewServer(icinga)
	defer ts.Close()

	l := newHostListener(t, ts, "labels:\n  project:\n    p-q2v7w: system\n")

	payload := WebhookPayload{Alerts: []Alert{
		clusterAlert("PodCrashLooping", "xraydmz2", "p-q2v7w"),
		clusterAlert("PodPending", "xraydmz2", "p-q2v7w"),
		clusterAlert("Watchdog", "xraydmz3", ""),
		{Status: "firing", Labels: map[string]string{"alertname": "NoCluster", "severity": "warning"}},
	}}

	if err := l.manageIcingaResults(context.Background(), payload); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"xraydmz2-system", "xraydmz3"} {
		host, ok := icinga.hosts[name]
		if !ok {
			t.Fatalf("host %s was not created; requests: %v", name, icinga.requests)
		}

		vars, _ := host["vars"].(map[string]any)
		if vars["bridge_uuid"] != "unittest" {
			t.Errorf("host %s: unexpected vars %v", name, vars)
		}
	}

	if vars, _ := icinga.hosts["xraydmz2-system"]["vars"].(map[string]any); vars["project"] != "system" || vars["project_source"] != "p-q2v7w" || vars["cluster"] != "xraydmz2" {
		t.Errorf("unexpected vars %v", vars)
	}

	creates := 0

	for _, req := range icinga.requests {
		if strings.HasPrefix(req, "PUT /v1/objects/hosts/") {
			creates++
		}
	}

	if creates != 2 {
		t.Errorf("want 2 host creations, got %d: %v", creates, icinga.requests)
	}

	wantHosts := map[string]int{"xraydmz2-system": 2, "xraydmz3": 1, "unittest": 1}
	gotHosts := map[string]int{}

	for name, svc := range icinga.services {
		host, _, _ := strings.Cut(name, "!")
		gotHosts[host]++

		if svc["host_name"] != host {
			t.Errorf("service %s has host_name %v", name, svc["host_name"])
		}
	}

	for host, n := range wantHosts {
		if gotHosts[host] != n {
			t.Errorf("host %s: want %d services, got %d (%v)", host, n, gotHosts[host], gotHosts)
		}
	}

	if len(icinga.results) != 4 {
		t.Fatalf("want 4 results, got %d", len(icinga.results))
	}

	if filter, _ := icinga.results[0]["filter"].(string); !strings.HasPrefix(filter, `host.name=="xraydmz2-system" && service.name=="PodCrashLooping_`) {
		t.Errorf("unexpected filter %q", filter)
	}

	// A second notification needs no host lookups any more
	icinga.requests = nil

	if err := l.manageIcingaResults(context.Background(), payload); err != nil {
		t.Fatal(err)
	}

	for _, req := range icinga.requests {
		if strings.Contains(req, "/v1/objects/hosts/") {
			t.Errorf("unexpected host request %s", req)
		}
	}
}

func TestManageIcingaResults_ExplicitHostWins(t *testing.T) {
	icinga := newFakeIcinga()
	icinga.hosts["MyHost"] = map[string]any{"name": "MyHost"}

	ts := httptest.NewServer(icinga)
	defer ts.Close()

	l := newHostListener(t, ts, "")

	alert := clusterAlert("Down", "xraydmz2", "p-q2v7w")
	alert.Labels["icinga_use_host"] = "MyHost"

	if err := l.manageIcingaResults(context.Background(), WebhookPayload{Alerts: []Alert{alert}}); err != nil {
		t.Fatal(err)
	}

	if len(icinga.hosts) != 1 {
		t.Fatalf("no host must be created, got %v", icinga.hosts)
	}

	for name := range icinga.services {
		if !strings.HasPrefix(name, "MyHost!Down_") {
			t.Errorf("unexpected service %s", name)
		}
	}
}

func TestManageIcingaResults_ContinuesAfterFailure(t *testing.T) {
	icinga := newFakeIcinga()

	ts := httptest.NewServer(icinga)
	defer ts.Close()

	l := newHostListener(t, ts, "")

	payload := WebhookPayload{Alerts: []Alert{
		// The fallback host "unittest" does not exist, so this one fails
		{Status: "firing", Labels: map[string]string{"alertname": "NoCluster", "severity": "warning"}},
		clusterAlert("PodPending", "xraydmz2", "p-1"),
	}}

	err := l.manageIcingaResults(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "alert NoCluster") {
		t.Fatalf("want an error for NoCluster, got %v", err)
	}

	if len(icinga.results) != 1 {
		t.Fatalf("the second alert must still be processed, got %d results", len(icinga.results))
	}
}

func TestAuthHandler_MultipleTokens(t *testing.T) {
	handler := authHandler(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }, "cluster-a", "cluster-b")

	for token, want := range map[string]int{"cluster-a": 200, "cluster-b": 200, "cluster-c": 401, "": 401} {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != want {
			t.Errorf("token %q: want %d, got %d", token, want, w.Code)
		}
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
