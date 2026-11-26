// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"testing"
)

func TestCLIConfig(t *testing.T) {
	expectedCert := "testdata/selfsigned.cert.pem"
	expectedKey := "testdata/selfsigned.key.pem"

	cli := CLI{
		TLSKeyPath:   expectedKey,
		TLSCertPath:  expectedCert,
		IcingaCAFile: expectedCert,
		BearerToken:  []string{"token"},
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

func TestCLIConfig_Validation(t *testing.T) {
	testCases := map[string]struct {
		have    CLI
		wantErr string
	}{
		"blank tokens": {
			have:    CLI{BearerToken: []string{" ", ""}},
			wantErr: "at least one non-empty bearer token is required",
		},
		"mapping without labels": {
			have:    CLI{BearerToken: []string{"t"}, HostMappingFile: "mapping.yaml"},
			wantErr: "a host mapping file requires --host-labels",
		},
		"labels are trimmed": {
			have: CLI{BearerToken: []string{"t", " u "}, HostLabels: []string{" cluster", "", "project "}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := NewConfigFromCLI(&tc.have)

			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("want error %q, got %v", tc.wantErr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if len(got.BearerTokens) != 2 || got.BearerTokens[1] != "u" {
				t.Errorf("unexpected tokens %q", got.BearerTokens)
			}

			if len(got.HostLabels) != 2 || got.HostLabels[0] != "cluster" || got.HostLabels[1] != "project" {
				t.Errorf("unexpected host labels %q", got.HostLabels)
			}
		})
	}
}
