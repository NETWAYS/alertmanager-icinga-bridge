// SPDX-License-Identifier: BSD-3-Clause

// Package hosts derives Icinga host names from alert labels and creates
// those hosts in Icinga on demand.
package hosts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// MappingScopeSeparator separates the values of the preceding host labels
// from the value itself in scoped mapping keys, e.g. "xraydmz2/p-q2v7w".
const MappingScopeSeparator = "/"

var invalidNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Mapping translates label values before they become part of a host name.
//
//	defaults:
//	  project: cluster          # used when an alert has no "project" label
//	labels:
//	  project:
//	    p-q2v7w: system         # for any cluster
//	    xraydmz2/p-abc12: infra # only for cluster "xraydmz2"
//
// A key may be scoped by the (unmapped) values of the host labels that come
// before the label, joined by "/". Scoped keys take precedence.
type Mapping struct {
	Defaults map[string]string            `yaml:"defaults"`
	Labels   map[string]map[string]string `yaml:"labels"`
}

// LoadMapping reads a YAML (or JSON) mapping file.
func LoadMapping(path string) (Mapping, error) {
	var m Mapping

	data, err := os.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("could not read host mapping file: %w", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	err = dec.Decode(&m)
	if err != nil && !errors.Is(err, io.EOF) {
		return m, fmt.Errorf("could not parse host mapping file %s: %w", path, err)
	}

	return m, nil
}

// Target is the Icinga host an alert belongs to.
type Target struct {
	// Name is the Icinga host name.
	Name string
	// Labels are the host labels that were used, after mapping.
	Labels map[string]string
	// Sources are the original label values for those labels that were mapped.
	Sources map[string]string
}

// Resolver derives host names from alert labels.
type Resolver struct {
	labels    []string
	separator string
	prefix    string
	path      string

	mu      sync.RWMutex
	mapping Mapping
}

// NewResolver returns a Resolver for the given labels. The mapping file is
// optional; an empty path means no mapping.
func NewResolver(labels []string, separator, prefix, mappingPath string) (*Resolver, error) {
	r := &Resolver{
		labels:    labels,
		separator: separator,
		prefix:    prefix,
		path:      mappingPath,
	}

	err := r.Reload()
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Enabled reports whether any host labels are configured.
func (r *Resolver) Enabled() bool {
	return r != nil && len(r.labels) > 0
}

// Reload re-reads the mapping file. On error, the previous mapping is kept.
func (r *Resolver) Reload() error {
	if r.path == "" {
		return nil
	}

	m, err := LoadMapping(r.path)
	if err != nil {
		return err
	}

	for label := range m.Labels {
		if !slices.Contains(r.labels, label) {
			return fmt.Errorf("host mapping file %s maps label %q, which is not one of the host labels %v", r.path, label, r.labels)
		}
	}

	for label := range m.Defaults {
		if !slices.Contains(r.labels, label) {
			return fmt.Errorf("host mapping file %s has a default for label %q, which is not one of the host labels %v", r.path, label, r.labels)
		}
	}

	r.mu.Lock()
	r.mapping = m
	r.mu.Unlock()

	return nil
}

// Path returns the mapping file path.
func (r *Resolver) Path() string {
	return r.path
}

// Resolve returns the host for the given alert labels. It returns false if
// the first host label is neither set nor has a default; other missing labels
// are left out of the name.
func (r *Resolver) Resolve(labels map[string]string) (Target, bool) {
	if !r.Enabled() {
		return Target{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	t := Target{
		Labels:  make(map[string]string, len(r.labels)),
		Sources: map[string]string{},
	}

	parts := make([]string, 0, len(r.labels))
	scope := make([]string, 0, len(r.labels))

	for i, label := range r.labels {
		raw := labels[label]
		if raw == "" {
			raw = r.mapping.Defaults[label]
		}

		if raw == "" {
			if i == 0 {
				return Target{}, false
			}

			continue
		}

		value := r.lookup(label, scope, raw)
		scope = append(scope, raw)

		part := sanitize(value)
		if part == "" {
			if i == 0 {
				return Target{}, false
			}

			continue
		}

		t.Labels[label] = value
		if value != raw {
			t.Sources[label] = raw
		}

		parts = append(parts, part)
	}

	t.Name = r.prefix + strings.Join(parts, r.separator)

	return t, true
}

// lookup maps a label value, preferring the key scoped by the preceding values.
func (r *Resolver) lookup(label string, scope []string, raw string) string {
	values, ok := r.mapping.Labels[label]
	if !ok {
		return raw
	}

	if len(scope) > 0 {
		key := strings.Join(append(append([]string{}, scope...), raw), MappingScopeSeparator)
		if v, ok := values[key]; ok {
			return v
		}
	}

	if v, ok := values[raw]; ok {
		return v
	}

	return raw
}

// sanitize replaces characters that are awkward in Icinga object names,
// URLs and filters ("!" is not allowed in host names at all).
func sanitize(s string) string {
	return strings.Trim(invalidNameChars.ReplaceAllString(strings.TrimSpace(s), "_"), "_")
}
