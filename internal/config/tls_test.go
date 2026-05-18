// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"crypto/tls"
	"reflect"
	"testing"
)

func TestTLSConfigEmpty(t *testing.T) {
	configTLSConfig := TLSConfig{
		InsecureSkipVerify: true,
	}

	expected := &tls.Config{
		InsecureSkipVerify: configTLSConfig.InsecureSkipVerify,
	}

	actual, err := NewTLSConfig(&configTLSConfig)
	if err != nil {
		t.Error("did not expect error", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Error("\nActual: ", actual, "\nExpected: ", expected)
	}
}

func TestTLSConfig(t *testing.T) {
	testcert := "testdata/selfsigned.cert.pem"
	testkey := "testdata/selfsigned.key.pem"

	configTLSConfig := TLSConfig{
		InsecureSkipVerify: true,
		ServerName:         "Test",
		CAFile:             testcert,
		CertFile:           testcert,
		KeyFile:            testkey,
	}

	actual, err := NewTLSConfig(&configTLSConfig)
	if err != nil {
		t.Error("did not expect error", err)
	}

	actualcert, err := tls.LoadX509KeyPair(testcert, testkey)
	if err != nil {
		t.Error("did not expect error", err)
	}

	cert, err := actual.GetClientCertificate(nil)
	if err != nil {
		t.Error("did not expect error", err)
	}
	if !reflect.DeepEqual(cert, &actualcert) {
		t.Error("\nActual: ", cert, "\nExpected: ", actualcert)
	}
}
