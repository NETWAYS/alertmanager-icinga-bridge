// SPDX-License-Identifier: BSD-3-Clause

package gc

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

func testConfig(url string) *config.Config {
	return &config.Config{
		IcingaURL:        []string{url},
		IcingaHostname:   "unittest",
		IcingaPassword:   "password",
		IcingaUser:       "username",
		HeartbeatService: "heartbeat",
	}
}

func testServerForDelete() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/objects/service/unittest!svc":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestGCRemoveService_WithRemoved(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName: "unittest",
		Name:     "svc",
		Vars: icinga2.Vars{
			"keep_for": 123.456,
		},
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "\"Successfully removed service from Icinga\" component=gc service=unittest!svc"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGCRemoveService_WithHeartbeatNoDowntime(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName: "unittest",
		Name:     "svc",
		Vars: icinga2.Vars{
			"keep_for":        20.0,
			"label_heartbeat": "300s",
		},
		LastStateChange: 1770000000.0,
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "Skipping heartbeat and not downtimed service"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n %v, got:\n %v", expected, actual)
	}
}

func TestGCRemoveService_WithHeartbeatDowntime(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName: "unittest",
		Name:     "svc",
		Vars: icinga2.Vars{
			"keep_for":        20.0,
			"label_heartbeat": "300s",
		},
		LastStateChange: 1770000000.0,
		DowntimeDepth:   1,
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "Deleting service at Icinga API"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected:\n %v, got:\n %v", expected, actual)
	}
}

func TestGCRemoveService_WithSkippedNotOK(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName: "unittest",
		Name:     "svc",
		State:    2,
		Vars: icinga2.Vars{
			"keep_for": 20.0,
		},
		LastStateChange: 1770000000.0,
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "Skipping not-OK services that are not heartbeats"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGCRemoveService_WithNoKeepFor(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName:        "unittest",
		Name:            "svc",
		State:           0,
		LastStateChange: 1770000000.0,
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "Deleting service at Icinga API"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGCRemoveService_WithInvalidKeepFor(t *testing.T) {
	ts := testServerForDelete()
	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	gc := NewGarbageCollector(config, logger, icingaClient)

	svc := icinga2.Service{
		HostName: "unittest",
		Name:     "svc",
		State:    0,
		Vars: icinga2.Vars{
			"keep_for": "FOOBAR",
		},
		LastStateChange: 1770000000.0,
	}

	actualErr := gc.removeServiceIfRequired(context.Background(), svc)

	if actualErr != nil {
		t.Errorf("expected no error got %v", actualErr)
	}

	actual := buf.String()
	expected := "Deleting service at Icinga API"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}
