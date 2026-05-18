// SPDX-License-Identifier: BSD-3-Clause

package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

func loadTestdata(filepath string) []byte {
	data, _ := os.ReadFile(filepath)
	return data
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func testConfig(url string) *config.Config {
	return &config.Config{
		IcingaURL:      []string{url},
		ID:             "unittest",
		IcingaHostname: "unittest",
	}
}

func TestAuthHandler_WithOK(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	var handler http.Handler = http.HandlerFunc(h)
	handler = authHandler(handler, "foobar123")
	w := httptest.NewRecorder()

	reqNoType, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	reqNoType.Header.Add("Authorization", "missingtype")

	handler.ServeHTTP(w, reqNoType)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	reqInvalid, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	reqInvalid.Header.Add("Authorization", "Bearer WRONG")

	handler.ServeHTTP(w, reqInvalid)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_WithFail(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	var handler http.Handler = http.HandlerFunc(h)
	handler = authHandler(handler, "foobar123")
	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Add("Authorization", "Bearer foobar123")

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListener_handleHealthy_WithOK(t *testing.T) {
	l := &Listener{
		logger: testLogger(),
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	l.handleHealthy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	actual := w.Body.String()
	expected := `{"status": "ok"}`

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGenerateServiceName(t *testing.T) {
	alertF := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "example",
		},
	}

	alertR := Alert{
		Status: "resolved",
		Labels: map[string]string{
			"alertname": "example",
		},
	}

	expected := "example_5ffb3b3c756e0110"

	actualF := generateServiceName("unittest", alertF)
	actualR := generateServiceName("unittest", alertR)

	if actualF != expected {
		t.Fatalf("expected %v, got %v", expected, actualF)
	}

	if actualF != actualR {
		t.Fatalf("expected firing and resolved alerts to have same name. firing: %v, resolved: %v", actualF, actualR)
	}
}

func TestManageIcingaService(t *testing.T) {
	datasetHost := "testdata/host.json"
	datasetService := "testdata/service.json"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/objects/hosts/unittest":
			w.WriteHeader(http.StatusOK)
			w.Write(loadTestdata(datasetHost))
		case "/v1/objects/hosts/unittest!heartbeat":
			w.WriteHeader(http.StatusOK)
			w.Write(loadTestdata(datasetService))
		case "/v1/actions/process-check-result/":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))

	defer ts.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := testConfig(ts.URL)

	icingaClient := icinga2.NewClient(config, logger)

	l := &Listener{
		logger:               logger,
		config:               config,
		icingaClient:         icingaClient,
		serviceNameValidator: serviceNamePattern,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "HighCPUUsage",
			"instance":  "server-01",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"summary":     "CPU usage > 90% on server01",
			"description": "The CPU has been above 90% for the last 5 minutes.",
		},
		GeneratorURL: "http://prometheus.example.com/",
		Fingerprint:  "a1b2c3d4e5f6g7h8i9j0",
	}

	payload := WebhookPayload{
		Status:   "firing",
		Receiver: "webhook",
		GroupLabels: map[string]string{
			"alertname": "HighCPUUsage",
		},
		CommonLabels: map[string]string{
			"team": "ops",
		},
		ExternalURL: "http://alertmanager.internal",
		Alerts:      []Alert{alert},
	}
	err := l.manageIcingaService(context.Background(), payload)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}

	actual := buf.String()
	expected := "Managed Icinga service"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestPrepareService_WithNoHeartbeat(t *testing.T) {
	config := testConfig("")

	config.StaticServiceVars = map[string]string{
		"foo": "bar",
	}

	l := &Listener{
		logger: testLogger(),
		config: config,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "example",
		},
		Annotations: map[string]string{
			"summary":     "example",
			"runbook_url": "unittest",
		},
	}

	svc := l.prepareService("serviceName", "displayName", alert, 0, time.Duration(0))

	if svc.NotesURL != "unittest" {
		t.Fatalf("expected %v, got %v", "unittest", svc.NotesURL)
	}
}

func TestPrepareService_WithHeartbeat(t *testing.T) {
	config := testConfig("")

	config.StaticServiceVars = map[string]string{
		"foo": "bar",
	}

	l := &Listener{
		logger: testLogger(),
		config: config,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "example",
		},
		Annotations: map[string]string{
			"summary":     "example",
			"runbook_url": "unittest",
		},
	}

	svc := l.prepareService("serviceName", "displayName", alert, 0, time.Duration(100))

	if svc.EnableActiveChecks != true {
		t.Fatalf("expected EnableActiveChecks to be true got false")
	}

	if svc.RetryInterval != 1.1e-07 {
		t.Fatalf("expected %v, got %v", "1.1e-07", svc.RetryInterval)
	}
}

func TestPrepareService_WithZoneHostTemplate(t *testing.T) {
	config := testConfig("")

	config.StaticServiceVars = map[string]string{
		"foo": "bar",
	}

	l := &Listener{
		logger: testLogger(),
		config: config,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname":           "example",
			"icinga_use_zone":     "myZone",
			"icinga_use_template": "myTemplate",
			"icinga_use_host":     "myHost",
		},
		Annotations: map[string]string{
			"summary":     "example",
			"runbook_url": "unittest",
		},
	}

	svc := l.prepareService("serviceName", "displayName", alert, 0, time.Duration(100))

	if svc.HostName != "myHost" {
		t.Fatalf("expected %v, got %v", "myHost", svc.HostName)
	}
	if svc.Zone != "myZone" {
		t.Fatalf("expected %v, got %v", "myZone", svc.Zone)
	}
}

func TestGeneratePluginOutput_WithPluginOutputByStates(t *testing.T) {
	config := testConfig("")

	config.PluginOutputAnnotations = []string{"barfoo", "foobar"}
	config.PluginOutputByStates = true

	l := &Listener{
		logger: testLogger(),
		config: config,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "example",
		},
		Annotations: map[string]string{
			"barfoo_ok": "UNITTEST",
			"foobar_ok": "NOTME",
			"notme":     "NOTME",
		},
	}

	actual := l.generatePluginOutput(alert, 0)
	expected := "UNITTEST"

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGeneratePluginOutput_WithoutPluginOutputByStates(t *testing.T) {
	config := testConfig("")

	config.PluginOutputAnnotations = []string{"barfoo", "foobar"}
	config.PluginOutputByStates = false

	l := &Listener{
		logger: testLogger(),
		config: config,
	}

	alert := Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "example",
		},
		Annotations: map[string]string{
			"barfoo_ok": "NOT",
			"foobar_ok": "NOTME",
			"notme":     "NOTME",
			"foobar":    "UNITTEST",
		},
	}

	actual := l.generatePluginOutput(alert, 0)
	expected := "UNITTEST"

	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestServiceNamePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{
			input:    "service",
			expected: true,
		},
		{
			input:    "a",
			expected: true,
		},
		{
			input:    "1234",
			expected: true,
		},
		{
			input:    "abc DEF123_.:,",
			expected: true,
		},
		{
			input:    "ex.ampl.e",
			expected: true,
		},
		{
			input:    "?&",
			expected: false,
		},
		{
			input:    "abc\r\n",
			expected: false,
		},
		{
			input:    "äöü",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if tc.expected != serviceNamePattern.MatchString(tc.input) {
				t.Fatalf("expected %v, to match %v, got %v", tc.input, tc.expected, !tc.expected)
			}
		})
	}
}

func TestSeverityToExitCode(t *testing.T) {
	severityLevels := map[string]int{
		"critical": 2,
		"warning":  1,
		"info":     0,
	}

	tests := []struct {
		name     string
		status   string
		severity string
		want     int
	}{
		{"firing-critical", "firing", "critical", 2},
		{"firing-warning", "firing", "warning", 1},
		{"firing-info", "firing", "info", 0},
		{"firing-unknown-severity", "firing", "unknown", 3},
		{"firing-mixed-case", "firing", "CrItIcAl", 2},
		{"resolved-any-severity", "resolved", "critical", 0},
		{"resolved-empty-severity", "resolved", "", 0},
		{"other-status", "pending", "critical", 3},
		{"empty-status", "", "critical", 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := severityToExitCode(tt.status, tt.severity, severityLevels); got != tt.want {
				t.Errorf("severityToExitCode(%q, %q) = %d, want %d",
					tt.status, tt.severity, got, tt.want)
			}
		})
	}
}

func TestMapToStableString(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want string
	}{
		{
			name: "empty map",
			in:   map[string]string{},
			want: "",
		},
		{
			name: "single key",
			in:   map[string]string{"foo": "bar"},
			want: "foo:bar ",
		},
		{
			name: "multiple keys – sorted output",
			in: map[string]string{
				"z":   "last",
				"a":   "first",
				"mid": "middle",
			},
			want: "a:first mid:middle z:last ",
		},
		{
			name: "severity key is ignored",
			in: map[string]string{
				"severity": "critical",
				"app":      "myapp",
				"env":      "prod",
			},
			want: "app:myapp env:prod ",
		},
		{
			name: "mixed order with severity",
			in: map[string]string{
				"b":        "2",
				"severity": "warning",
				"a":        "1",
				"c":        "3",
				"irrelev":  "x",
			},
			want: "a:1 b:2 c:3 irrelev:x ",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := mapToStableString(tt.in); got != tt.want {
				t.Errorf("mapToStableString() = %q, want %q", got, tt.want)
			}
		})
	}
}

var mapIcingaVariableTest = map[string]struct {
	iK  string
	iV  string
	oK  string
	oV  any
	err error
}{
	"not mapped":    {"foo", "bar", "foo", "bar", errNotAMappingKey},
	"mapped number": {"icinga_number_foo", "42", "foo", 42, nil},
	"mapped string": {"icinga_string_foo", "bar", "foo", "bar", nil},
	"unknown":       {"icinga_unknown_foo", "bar", "", nil, errUnknownMappingType},
}

func TestMapIcingaVariable(t *testing.T) {
	for name, test := range mapIcingaVariableTest {
		t.Run(name, func(t *testing.T) {
			k, v, err := mapIcingaVariable(test.iK, test.iV)
			if err != test.err {
				t.Errorf("expected error %v, got %v", test.err, err)
			}
			if k != test.oK {
				t.Errorf("expected key %q, got %q", test.oK, k)
			}
			if v != test.oV {
				t.Errorf("expected value %q, got %q", test.oV, v)
			}
		})
	}
}

func TestMapIcingaVariables(t *testing.T) {
	vars := make(icinga2.Vars)
	kv := map[string]string{
		"a":                "a",
		"icinga_number_b":  "42",
		"icinga_string_c":  "c",
		"icinga_unknown_d": "d",
		"icinga_number_e":  "e",
	}
	vars = mapIcingaVariables(vars, kv, "pre_")
	expected := icinga2.Vars{
		"pre_a":                "a",
		"pre_icinga_number_b":  "42",
		"pre_icinga_string_c":  "c",
		"pre_icinga_unknown_d": "d",
		"pre_icinga_number_e":  "e",
		"b":                    42,
		"c":                    "c",
	}
	if !reflect.DeepEqual(vars, expected) {
		t.Errorf("mapIcingaVariables() = %+v, want %+v", vars, expected)
	}
}
