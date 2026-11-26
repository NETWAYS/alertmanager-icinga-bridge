// SPDX-License-Identifier: Apache-2.0

package icinga2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/config"
)

func prettyPrint(i any) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

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
		IcingaHostname: "unittest",
		IcingaPassword: "password",
		IcingaUser:     "username",
	}
}

func jsonEqual(a, b any) bool {
	ba, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ba) == string(bb)
}

func TestServiceFullName(t *testing.T) {
	svc := Service{
		Name:     "heartbeat",
		HostName: "Alertmanager",
	}

	expected := "Alertmanager!heartbeat"

	if svc.FullName() != expected {
		t.Fatalf("expected:\n %s \ngot:\n %s", expected, svc.FullName())
	}
}

func TestGetHost_WithInvalidReponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<invalid JSON>`))
	}))

	defer server.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(testConfig(server.URL), logger)

	_, err := client.GetHost(context.Background(), "unittest")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	actual := buf.String()
	expected := "Response body: <invalid JSON>"

	if !strings.Contains(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestGetHost(t *testing.T) {
	datasetHost := "testdata/host.json"

	tests := map[string]struct {
		server      *httptest.Server
		args        []string
		expected    Host
		expectedErr error
	}{
		"getHost": {
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(loadTestdata(datasetHost))
			})),
			expectedErr: nil,
			expected: Host{
				Name:        "unittest",
				DisplayName: "unittest",
			},
		},
		"noSuchHost": {
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":404,"status":"No objects found."}`))
			})),
			expectedErr: ErrNotFound,
			expected:    Host{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer test.server.Close()

			client := NewClient(testConfig(test.server.URL), testLogger())

			actual, err := client.GetHost(context.Background(), "unittest")

			if err != test.expectedErr {
				t.Fatalf("expected:\n %s \ngot:\n %s", prettyPrint(test.expectedErr), prettyPrint(err))
			}

			if !reflect.DeepEqual(actual, test.expected) {
				t.Fatalf("expected:\n %s \ngot:\n %s", prettyPrint(test.expected), prettyPrint(actual))
			}
		})
	}
}

func TestGetService(t *testing.T) {
	datasetService := "testdata/service.json"

	tests := map[string]struct {
		server      *httptest.Server
		args        []string
		expected    Service
		expectedErr error
	}{
		"getService": {
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(loadTestdata(datasetService))
			})),
			expectedErr: nil,
			expected: Service{
				Name:               "heartbeat",
				DisplayName:        "heartbeat",
				HostName:           "Alertmanager",
				CheckCommand:       "dummy",
				CheckInterval:      10,
				LastStateChange:    123.456,
				MaxCheckAttempts:   3,
				RetryInterval:      60,
				EnableActiveChecks: true,
				State:              2,
				Vars: Vars{
					"dummy_state":                 2,
					"dummy_text.deprecated":       false,
					"dummy_text.name":             "\u003canonymous\u003e",
					"dummy_text.side_effect_free": false,
					"dummy_text.nest.foo[0]":      1,
					"dummy_text.type":             "Function",
				},
			},
		},
		"noSuchService": {
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":404,"status":"No objects found."}`))
			})),
			expectedErr: ErrNotFound,
			expected:    Service{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer test.server.Close()

			client := NewClient(testConfig(test.server.URL), testLogger())

			actual, err := client.GetService(context.Background(), "heartbeat")

			if err != test.expectedErr {
				t.Fatalf("expected err:\n %s \ngot:\n %s", test.expectedErr, err)
			}

			if prettyPrint(test.expected) != prettyPrint(actual) {
				t.Fatalf("expected:\n %s \ngot:\n %s", prettyPrint(test.expected), prettyPrint(actual))
			}
		})
	}
}

func TestCreateAndDeleteHost(t *testing.T) {
	var got map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v1/objects/hosts/new":
			_ = json.NewDecoder(r.Body).Decode(&got)
		case r.Method == http.MethodPut && r.URL.Path == "/v1/objects/hosts/exists":
			http.Error(w, `{"results":[{"code":500,"errors":["Object 'exists' already exists."],"status":"Object could not be created."}]}`, http.StatusInternalServerError)
		case r.Method == http.MethodPut:
			http.Error(w, `{"results":[{"code":500,"errors":["Import references unknown template: 'nope'"]}]}`, http.StatusInternalServerError)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/objects/hosts/new" && r.URL.Query().Get("cascade") == "1":
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/objects/hosts/forbidden":
			http.Error(w, `{"error":403,"status":"No objects found."}`, http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(testConfig(ts.URL), testLogger())
	ctx := context.Background()

	if err := c.CreateHost(ctx, "new", []string{"generic-host"}, map[string]any{"vars": map[string]any{"a": "b"}}); err != nil {
		t.Fatal(err)
	}

	if templates, _ := got["templates"].([]any); len(templates) != 1 || templates[0] != "generic-host" {
		t.Errorf("unexpected body %v", got)
	}

	if err := c.CreateHost(ctx, "exists", nil, nil); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("want ErrAlreadyExists, got %v", err)
	}

	if err := c.CreateHost(ctx, "broken", []string{"nope"}, nil); err == nil || !strings.Contains(err.Error(), "unknown template") {
		t.Errorf("want the Icinga error message, got %v", err)
	}

	if err := c.DeleteHost(ctx, "new"); err != nil {
		t.Fatal(err)
	}

	if err := c.DeleteHost(ctx, "gone"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}

	var statusErr *StatusError
	if err := c.DeleteHost(ctx, "forbidden"); !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusForbidden {
		t.Errorf("want a 403 StatusError, got %v", err)
	}
}
