package icinga2

import (
	"testing"
)

func TestExitStatusString(t *testing.T) {
	testCases := map[string]struct {
		subject    ExitStatus
		wantString string
	}{
		"ok": {
			subject:    ExitStatusOK,
			wantString: "OK",
		},
		"warn": {
			subject:    ExitStatusWarning,
			wantString: "WARNING",
		},
		"crit": {
			subject:    ExitStatusCritical,
			wantString: "CRITICAL",
		},
		"unknown": {
			subject:    ExitStatusUnknown,
			wantString: "UNKNOWN",
		},
		"custom": {
			subject:    ExitStatus(9001),
			wantString: "EXIT_9001",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if got := testCase.subject.String(); got != testCase.wantString {
				t.Errorf("ExitStatus.String() yielded wrong result: got=%q, want=%q", got, testCase.wantString)
			}
		})
	}
}

func TestExitStatusMarshalText(t *testing.T) {
	testCases := map[string]struct {
		subject    ExitStatus
		wantString string
	}{
		"ok": {
			subject:    ExitStatusOK,
			wantString: "0",
		},
		"warn": {
			subject:    ExitStatusWarning,
			wantString: "1",
		},
		"crit": {
			subject:    ExitStatusCritical,
			wantString: "2",
		},
		"unknown": {
			subject:    ExitStatusUnknown,
			wantString: "3",
		},
		"custom": {
			subject:    ExitStatus(9001),
			wantString: "9001",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			b, err := testCase.subject.MarshalText()
			if err != nil {
				t.Fatalf("ExitStatus.MarshalText() produced error: %v", err)
			}

			if got := string(b); got != testCase.wantString {
				t.Errorf("ExitStatus.MarshalText() yielded wrong result: got=%q, want=%q", got, testCase.wantString)
			}
		})
	}
}
