package api

import (
	"strings"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

// AlertFieldGetter is a visitor for alert instances with
// the intention to extract members.
type AlertFieldGetter interface {
	// GetField returns a field from the provided alert
	// based in the concrete implementation.
	GetField(a *Alert, es icinga2.ExitStatus) string
}

// AnnotationsGetter retrieves the first annotation value from alerts which
// match the given slice of keys.
type AnnotationsGetter []string

// GetField iterates over the [Alert.Annotations] and returns the value
// of the first entry which matches the slice of search keys or an empty
// string if no annotation matches any search key. The exit status is not
// used in the search.
func (ag AnnotationsGetter) GetField(a *Alert, _ icinga2.ExitStatus) string {
	for _, key := range ag {
		if annotation, ok := a.Annotations[key]; ok {
			return annotation
		}
	}

	return ""
}

// LabelsGetter retrieves the first label value from alerts which
// match the given slice of keys.
type LabelsGetter []string

// GetField iterates over the [Alert.Labels] and returns the value
// of the first entry which matches the slice of search keys or an empty
// string if no label matches any search key. The exit status is not
// used in the search.
func (lg LabelsGetter) GetField(a *Alert, _ icinga2.ExitStatus) string {
	for _, key := range lg {
		if label, ok := a.Labels[key]; ok {
			return label
		}
	}

	return ""
}

// AnnotationsPrefixGetter retrieves the first annotation value from alerts which
// match the given slice of keys combined with the string value of an [icinga2.ExitStatus]
type AnnotationsPrefixGetter []string

// GetField iterates over the [Alert.Annotations] and returns the value
// of the first entry which matches the slice of search keys or an empty
// string if no annotation matches any search key. Each search key is
// suffixed with an underscore and the provided [icinga2.ExitCode].
// If a lookup yields no result, the same search key is tried again without
// any suffix, similar to [AnnotationsGetter].
func (apg AnnotationsPrefixGetter) GetField(a *Alert, es icinga2.ExitStatus) string {
	suffix := "_" + strings.ToLower(es.String())

	for _, key := range apg {
		if annotation, ok := a.Annotations[key+suffix]; ok {
			return annotation
		} else if annotation, ok := a.Annotations[key]; ok {
			return annotation
		}
	}

	return ""
}
