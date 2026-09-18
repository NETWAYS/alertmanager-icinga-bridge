package api

import (
	"testing"
	"time"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"
)

type AlertFieldGetterGetFieldTestCase struct {
	haveInput []string
	haveAlert *Alert
	haveCode  icinga2.ExitStatus
	wantField string
}

func TestAnnotationsGetterGetField(t *testing.T) {
	annotations := map[string]string{
		"message":     "m-e-s-s-a-g-e",
		"summary":     "s-u-m-m-a-r-y",
		"description": "DeScRiPtIoN",
	}
	testCases := map[string]AlertFieldGetterGetFieldTestCase{
		"empty input": {
			haveAlert: newTestAlertAnnotations(alertStatusFiring, annotations),
			haveCode:  icinga2.ExitStatusUnknown,
		},
		"no match found": {
			haveInput: []string{"output", "stdout"},
			haveAlert: newTestAlertAnnotations(alertStatusResolved, annotations),
			haveCode:  icinga2.ExitStatusOK,
		},
		"match found": {
			haveInput: []string{"output", "summary", "stdout"},
			haveAlert: newTestAlertAnnotations(alertStatusFiring, annotations),
			haveCode:  icinga2.ExitStatusWarning,
			wantField: "s-u-m-m-a-r-y",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			subject := AnnotationsGetter(testCase.haveInput)
			if got := subject.GetField(testCase.haveAlert, testCase.haveCode); got != testCase.wantField {
				t.Errorf("AnnotationsGetter.GetField() yielded wrong result: got=%q, want=%q", got, testCase.wantField)
			}
		})
	}
}

func TestLabelsGetterGetField(t *testing.T) {
	labels := map[string]string{
		"severity": "s-e-v-e-r-i-t-y",
		"origin":   "o-r-i-g-i-n",
		"cause":    "CaUsE",
	}
	testCases := map[string]AlertFieldGetterGetFieldTestCase{
		"empty input": {
			haveAlert: newTestAlertLabels(alertStatusFiring, labels),
			haveCode:  icinga2.ExitStatusUnknown,
		},
		"no match found": {
			haveInput: []string{"recipient", "receiver"},
			haveAlert: newTestAlertLabels(alertStatusResolved, labels),
			haveCode:  icinga2.ExitStatusOK,
		},
		"match found": {
			haveInput: []string{"output", "origin", "stdout"},
			haveAlert: newTestAlertLabels(alertStatusFiring, labels),
			haveCode:  icinga2.ExitStatusWarning,
			wantField: "o-r-i-g-i-n",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			subject := LabelsGetter(testCase.haveInput)
			if got := subject.GetField(testCase.haveAlert, testCase.haveCode); got != testCase.wantField {
				t.Errorf("LabelsGetter.GetField() yielded wrong result: got=%q, want=%q", got, testCase.wantField)
			}
		})
	}
}

func TestAnnotationsPrefixGetterGetField(t *testing.T) {
	annotations := map[string]string{
		"message":          "default message",
		"message_critical": "stuff is on fire",
		"message_warning":  "not good, not terrible",
		"summary_critical": "all good things must come to an end",
		"summary_warning":  "uh-oh",
	}
	testCases := map[string]AlertFieldGetterGetFieldTestCase{
		"empty input": {
			haveAlert: newTestAlertAnnotations(alertStatusFiring, annotations),
			haveCode:  icinga2.ExitStatusUnknown,
		},
		"no prefix match found": {
			haveInput: []string{"description"},
			haveAlert: newTestAlertAnnotations(alertStatusResolved, annotations),
			haveCode:  icinga2.ExitStatusOK,
		},
		"no exit code match found": {
			haveInput: []string{"summary"},
			haveAlert: newTestAlertAnnotations(alertStatusResolved, annotations),
			haveCode:  icinga2.ExitStatusOK,
		},
		"prefix match found": {
			haveInput: []string{"message"},
			haveAlert: newTestAlertAnnotations(alertStatusFiring, annotations),
			haveCode:  icinga2.ExitStatusWarning,
			wantField: "not good, not terrible",
		},
		"fallback match found": {
			haveInput: []string{"message"},
			haveAlert: newTestAlertAnnotations(alertStatusFiring, annotations),
			haveCode:  icinga2.ExitStatusOK,
			wantField: "default message",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			subject := AnnotationsPrefixGetter(testCase.haveInput)
			if got := subject.GetField(testCase.haveAlert, testCase.haveCode); got != testCase.wantField {
				t.Errorf("AnnotationsPrefixGetter.GetField() yielded wrong result: got=%q, want=%q", got, testCase.wantField)
			}
		})
	}
}

func newTestAlertLabels(status string, labels map[string]string) *Alert {
	result := &Alert{
		Status:       status,
		Labels:       labels,
		StartsAt:     time.Unix(300, 0),
		EndsAt:       time.Unix(400, 0),
		GeneratorURL: "test://unit",
		Fingerprint:  "1234567890",
	}

	return result
}

func newTestAlertAnnotations(status string, annotations map[string]string) *Alert {
	result := &Alert{
		Status:       status,
		Annotations:  annotations,
		StartsAt:     time.Unix(200, 0),
		EndsAt:       time.Unix(500, 0),
		GeneratorURL: "test://unit",
		Fingerprint:  "0987654321",
	}

	return result
}
