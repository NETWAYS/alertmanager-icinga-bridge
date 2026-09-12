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
