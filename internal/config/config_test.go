// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"reflect"
	"testing"
)

func TestMergeSeverityLevels(t *testing.T) {
	tests := map[string]struct {
		custom     map[string]string
		wantLevels map[string]int
	}{
		"nil map defaults only": {
			custom: nil,
			wantLevels: map[string]int{
				"normal":   0,
				"warning":  1,
				"critical": 2,
			},
		},
		"empty map defaults only": {
			custom: map[string]string{},
			wantLevels: map[string]int{
				"normal":   0,
				"warning":  1,
				"critical": 2,
			},
		},
		"valid overrides": {
			custom: map[string]string{
				"Warning":  "5",
				"critical": "0",
				"custom":   "2",
			},
			wantLevels: map[string]int{
				"normal":   0,
				"warning":  3,
				"critical": 0,
				"custom":   2,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := mergeSeverityLevels(tt.custom)

			if !reflect.DeepEqual(got, tt.wantLevels) {
				t.Fatalf("expected %v, got %v", tt.wantLevels, got)
			}
		})
	}
}

func TestCLIConfig(t *testing.T) {
	expectedCert := "testdata/selfsigned.cert.pem"
	expectedKey := "testdata/selfsigned.key.pem"

	cli := CLI{
		TLSKeyPath:   expectedKey,
		TLSCertPath:  expectedCert,
		IcingaCAFile: expectedCert,
	}

	actual, err := NewConfigFromCLI(&cli)

	if err != nil {
		t.Fatal("did not expect error", err)
	}

	if actual.TLSCertPath != expectedCert {
		t.Fatalf("expected %v, got %v", expectedCert, actual.TLSCertPath)
	}

	if actual.TLSKeyPath != expectedKey {
		t.Fatalf("expected %v, got %v", expectedKey, actual.TLSKeyPath)
	}
}
