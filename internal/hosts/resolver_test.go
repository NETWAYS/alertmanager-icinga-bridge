// SPDX-License-Identifier: BSD-3-Clause

package hosts

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testMapping = `
defaults:
  project: cluster
labels:
  project:
    p-q2v7w: system
    xraydmz2/p-abc12: infra
    other/p-q2v7w: other-system
`

func writeMapping(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mapping.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func TestResolve(t *testing.T) {
	r, err := NewResolver([]string{"cluster", "project"}, "-", "", writeMapping(t, testMapping))
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		have        map[string]string
		wantOK      bool
		wantName    string
		wantLabels  map[string]string
		wantSources map[string]string
	}{
		"unmapped": {
			have:        map[string]string{"cluster": "xraydmz2", "project": "p-zzzzz"},
			wantOK:      true,
			wantName:    "xraydmz2-p-zzzzz",
			wantLabels:  map[string]string{"cluster": "xraydmz2", "project": "p-zzzzz"},
			wantSources: map[string]string{},
		},
		"mapped for any cluster": {
			have:        map[string]string{"cluster": "xraydmz2", "project": "p-q2v7w"},
			wantOK:      true,
			wantName:    "xraydmz2-system",
			wantLabels:  map[string]string{"cluster": "xraydmz2", "project": "system"},
			wantSources: map[string]string{"project": "p-q2v7w"},
		},
		"scoped mapping wins": {
			have:     map[string]string{"cluster": "other", "project": "p-q2v7w"},
			wantOK:   true,
			wantName: "other-other-system",
		},
		"scoped mapping only for its cluster": {
			have:     map[string]string{"cluster": "else", "project": "p-abc12"},
			wantOK:   true,
			wantName: "else-p-abc12",
		},
		"default for missing label": {
			have:     map[string]string{"cluster": "xraydmz2"},
			wantOK:   true,
			wantName: "xraydmz2-cluster",
		},
		"first label missing": {
			have:   map[string]string{"project": "p-q2v7w"},
			wantOK: false,
		},
		"invalid characters": {
			have:     map[string]string{"cluster": "my cluster!", "project": "a/b"},
			wantOK:   true,
			wantName: "my_cluster-a_b",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, ok := r.Resolve(tc.have)

			if ok != tc.wantOK {
				t.Fatalf("ok: want %v, got %v", tc.wantOK, ok)
			}

			if got.Name != tc.wantName {
				t.Errorf("name: want %q, got %q", tc.wantName, got.Name)
			}

			if tc.wantLabels != nil && !reflect.DeepEqual(got.Labels, tc.wantLabels) {
				t.Errorf("labels: want %v, got %v", tc.wantLabels, got.Labels)
			}

			if tc.wantSources != nil && !reflect.DeepEqual(got.Sources, tc.wantSources) {
				t.Errorf("sources: want %v, got %v", tc.wantSources, got.Sources)
			}
		})
	}
}

func TestResolve_OptionalLabelsAndPrefix(t *testing.T) {
	r, err := NewResolver([]string{"cluster", "project"}, ".", "k8s-", "")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := r.Resolve(map[string]string{"cluster": "xraydmz2"})
	if !ok || got.Name != "k8s-xraydmz2" {
		t.Fatalf("want k8s-xraydmz2, got %q (%v)", got.Name, ok)
	}

	got, ok = r.Resolve(map[string]string{"cluster": "xraydmz2", "project": "p-1"})
	if !ok || got.Name != "k8s-xraydmz2.p-1" {
		t.Fatalf("want k8s-xraydmz2.p-1, got %q (%v)", got.Name, ok)
	}
}

func TestResolve_Disabled(t *testing.T) {
	var nilResolver *Resolver

	if _, ok := nilResolver.Resolve(map[string]string{"cluster": "a"}); ok {
		t.Fatal("nil resolver must not resolve")
	}

	r, _ := NewResolver(nil, "-", "", "")
	if _, ok := r.Resolve(map[string]string{"cluster": "a"}); ok {
		t.Fatal("resolver without labels must not resolve")
	}
}

func TestReload(t *testing.T) {
	path := writeMapping(t, testMapping)

	r, err := NewResolver([]string{"cluster", "project"}, "-", "", path)
	if err != nil {
		t.Fatal(err)
	}

	// A broken file keeps the previous mapping
	_ = os.WriteFile(path, []byte("labels: [unclosed"), 0o600)

	if r.Reload() == nil {
		t.Fatal("expected an error for a broken mapping file")
	}

	if got, _ := r.Resolve(map[string]string{"cluster": "c", "project": "p-q2v7w"}); got.Name != "c-system" {
		t.Fatalf("previous mapping lost, got %q", got.Name)
	}

	_ = os.WriteFile(path, []byte(`{"labels": {"project": {"p-q2v7w": "renamed"}}}`), 0o600)

	if err := r.Reload(); err != nil {
		t.Fatal(err)
	}

	if got, _ := r.Resolve(map[string]string{"cluster": "c", "project": "p-q2v7w"}); got.Name != "c-renamed" {
		t.Fatalf("want c-renamed, got %q", got.Name)
	}
}

func TestLoadMapping_Errors(t *testing.T) {
	testCases := map[string]struct {
		content string
		wantErr string
	}{
		"unknown label":      {content: "labels:\n  team:\n    a: b\n", wantErr: `maps label "team"`},
		"unknown default":    {content: "defaults:\n  team: x\n", wantErr: `default for label "team"`},
		"unknown key":        {content: "label:\n  project: {}\n", wantErr: "field label not found"},
		"not a mapping file": {content: "- a\n- b\n", wantErr: "could not parse"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := NewResolver([]string{"cluster", "project"}, "-", "", writeMapping(t, tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}

	if _, err := NewResolver([]string{"cluster"}, "-", "", writeMapping(t, "")); err != nil {
		t.Fatalf("empty mapping file must be accepted, got %v", err)
	}
}
