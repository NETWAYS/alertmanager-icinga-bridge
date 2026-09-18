package icinga2

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActionMarshalJSON(t *testing.T) {
	testCases := map[string]struct {
		subject     *Action
		wantFixture string
	}{
		"minimal": {
			subject:     &Action{},
			wantFixture: "minimal",
		},
		"only non-empty": {
			subject: &Action{
				ExitStatus:   ExitStatusUnknown,
				PluginOutput: "test",
				TTL:          60,
				Filter:       "1 == 1",
				Type:         "service",
			},
			wantFixture: "nonempty",
		},
		"all fields": {
			subject: &Action{
				ExitStatus:      ExitStatusWarning,
				PluginOutput:    "test",
				TTL:             60,
				Filter:          "1 == 1",
				Type:            "service",
				PerformanceData: []string{"test_cases=1"},
				CheckCommand:    []string{"/usr/lib/nagios/plugins/check_test", "--test"},
				CheckSource:     "master",
				ExecutionStart:  "1234567890",
				ExecutionEnd:    "1234567890",
			},
			wantFixture: "full",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			golden := filepath.Join("testdata", "golden", "action", testCase.wantFixture+".json")
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("unable to load golden action model fixture: %v", err)
			}

			got, err := json.MarshalIndent(testCase.subject, "", "  ")
			if err != nil {
				t.Fatalf("Action.MarshalJSON() produced error: %v", err)
			}

			if want, got = bytes.TrimSpace(want), bytes.TrimSpace(got); bytes.Compare(want, got) != 0 {
				t.Errorf("Action.MarshalJSON() result did not match golden fixture: %s", got)
			}
		})
	}
}
