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
	"path/filepath"
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

func httpOperationError(w http.ResponseWriter, error string, code int) {
	status := OperationStatus{
		Code:   code,
		Status: error,
	}
	data := OperationResults{
		Results: []OperationStatus{status},
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(&data)
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

func TestProcessCheckResult(t *testing.T) {
	testCases := map[string]struct {
		haveAction Action
		wantErr    error
	}{
		"success": {
			haveAction: Action{
				ExitStatus: http.StatusOK,
				Filter:     "success",
			},
		},
		"simple operation error": {
			haveAction: Action{
				ExitStatus: http.StatusBadRequest,
				Filter:     "host-exit-code",
			},
			wantErr: errors.New("Invalid 'exit_status' for Host icinga2-master1.localdomain."),
		},
		"detailed operation error": {
			haveAction: Action{
				ExitStatus: http.StatusForbidden,
				Filter:     "forbidden",
			},
			wantErr: errors.New("User is not allowed to access the requested host object.\nUser permission does not grant access to this resource"),
		},
		"invalid JSON response": {
			haveAction: Action{
				ExitStatus: http.StatusBadRequest,
				Filter:     "malformed",
			},
			wantErr: errors.New("Unable to parse API error response: invalid character '}' looking for beginning of value"),
		},
		"not reachable": {
			haveAction: Action{
				ExitStatus: http.StatusServiceUnavailable,
			},
			wantErr: ErrNoEndpointReachable,
		},
		"not found": {
			haveAction: Action{
				ExitStatus: http.StatusNotFound,
				Filter:     "not-found",
			},
			wantErr: ErrNotFound,
		},
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			httpOperationError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := new(Action)
		err = json.Unmarshal(data, result)
		if err != nil {
			httpOperationError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if result.ExitStatus == 0 {
			httpOperationError(w, "Invalid 'exit_status'", http.StatusBadRequest)
			return
		} else if result.ExitStatus == http.StatusServiceUnavailable {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		name := result.Filter + ".json"
		fixture := filepath.Join("testdata", "process-check-result", name)
		file, err := os.Open(fixture)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.NotFound(w, r)
			} else {
				httpOperationError(w, err.Error(), http.StatusInternalServerError)
			}

			return
		}

		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			httpOperationError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(int(result.ExitStatus))
		http.ServeContent(w, r, name, stat.ModTime(), file)
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(handler))
			client := NewClient(testConfig(srv.URL), testLogger())
			defer srv.Close()

			got := client.ProcessCheckResult(context.Background(), testCase.haveAction)
			if testCase.wantErr == nil {
				if got != nil {
					t.Fatalf("ProcessCheckResult yielded error despite not expecting one: %s", got)
				}
			} else if got == nil {
				if testCase.wantErr != nil {
					t.Fatalf("ProcessCheckResult yielded no error despite expecting one")
				}
			} else if testCase.wantErr.Error() != got.Error() {
				if !errors.Is(got, testCase.wantErr) { // got might wrap several errors and wantErr might be one of them
					t.Fatalf("ProcessCheckResult error mismatch. want:\n %s \ngot:\n %s", testCase.wantErr, got)
				}
			}
		})
	}
}
