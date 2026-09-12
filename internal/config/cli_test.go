package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

func TestCLI(t *testing.T) {
	requiredArgs := []string{
		"--icinga-user=username",
		"--icinga-password=password",
		"--bearer-token=token",
		"--icinga-hostname=alertmanager.localhost",
		"--icinga-url=https://icinga.localhost:5665/",
		"--id=unit-test",
	}
	testCases := map[string]struct {
		haveCLI []string
		wantErr string
		wantCLI func(*testing.T, *CLI)
	}{
		"empty args": {
			wantErr: "missing flags: --bearer-token=STRING, --icinga-hostname=STRING, --icinga-password=STRING, --icinga-url=ICINGA-URL,..., --icinga-user=STRING, --id=STRING",
		},
		"minimal args": {
			haveCLI: requiredArgs,
			wantCLI: func(t *testing.T, got *CLI) {

				// REQUIRED

				wantID := "unit-test"
				if wantID != got.ID {
					t.Errorf("ID mismatch; want=%q, got=%q", wantID, got.ID)
				}

				wantIcingaURL := "https://icinga.localhost:5665/"
				if len(got.IcingaURL) != 1 && wantIcingaURL != got.IcingaURL[0] {
					t.Errorf("IcingaURL mismatch; want=%q, got=%q", wantIcingaURL, got.IcingaURL[0])
				}

				wantIcingaHostname := "alertmanager.localhost"
				if wantIcingaHostname != got.IcingaHostname {
					t.Errorf("IcingaHostname mismatch; want=%q, got=%q", wantIcingaHostname, got.IcingaHostname)
				}

				wantIcingaPassword := "password"
				if wantIcingaPassword != got.IcingaPassword {
					t.Errorf("IcingaPassword mismatch; want=%q, got=%q", wantIcingaPassword, got.IcingaPassword)
				}

				wantIcingaUser := "username"
				if wantIcingaUser != got.IcingaUser {
					t.Errorf("IcingaUser mismatch; want=%q, got=%q", wantIcingaUser, got.IcingaUser)
				}

				wantBearerToken := "token"
				if wantBearerToken != got.BearerToken {
					t.Errorf("BearerToken mismatch; want=%q, got=%q", wantBearerToken, got.BearerToken)
				}

				// DEFAULT

				wantLoglevel := LogLevel(slog.LevelInfo)
				if wantLoglevel != got.Loglevel {
					t.Errorf("Loglevel mismatch; want=%v, got=%v", wantLoglevel, got.Loglevel)
				}

				wantDisableKeepAlives := false
				if wantDisableKeepAlives != got.DisableKeepAlives {
					t.Errorf("DisableKeepAlives mismatch; want=%t, got=%t", wantDisableKeepAlives, got.DisableKeepAlives)
				}

				wantDisplayNameAsServiceName := false
				if wantDisplayNameAsServiceName != got.DisplayNameAsServiceName {
					t.Errorf("DisplayNameAsServiceName mismatch; want=%t, got=%t", wantDisplayNameAsServiceName, got.DisplayNameAsServiceName)
				}

				wantIcingaInsecureTLS := false
				if wantIcingaInsecureTLS != got.IcingaInsecureTLS {
					t.Errorf("IcingaInsecureTLS mismatch; want=%t, got=%t", wantIcingaInsecureTLS, got.IcingaInsecureTLS)
				}

				wantGCInterval := 15 * time.Minute
				if wantGCInterval != got.GCInterval {
					t.Errorf("GCInterval mismatch; want=%q, got=%q", wantGCInterval, got.GCInterval)
				}

				wantHeartbeatInterval := 1 * time.Minute
				if wantHeartbeatInterval != got.HeartbeatInterval {
					t.Errorf("HeartbeatInterval mismatch; want=%q, got=%q", wantHeartbeatInterval, got.HeartbeatInterval)
				}

				wantHeartbeatService := "heartbeat"
				if wantHeartbeatService != got.HeartbeatService {
					t.Errorf("HeartbeatService mismatch; want=%q, got=%q", wantHeartbeatService, got.HeartbeatService)
				}

				wantListenAddr := "127.0.0.1:8888"
				if wantListenAddr != got.ListenAddr {
					t.Errorf("ListenAddr mismatch; want=%q, got=%q", wantListenAddr, got.ListenAddr)
				}

				wantCheckCommand := "dummy"
				if wantCheckCommand != got.CheckCommand {
					t.Errorf("CheckCommand mismatch; want=%q, got=%q", wantCheckCommand, got.CheckCommand)
				}

				wantActiveChecks := false
				if wantActiveChecks != got.ActiveChecks {
					t.Errorf("ActiveChecks mismatch; want=%t, got=%t", wantActiveChecks, got.ActiveChecks)
				}

				wantPluginOutputByStates := false
				if wantPluginOutputByStates != got.PluginOutputByStates {
					t.Errorf("PluginOutputByStates mismatch; want=%t, got=%t", wantPluginOutputByStates, got.PluginOutputByStates)
				}

				wantMaxCheckAttempts := 1
				if wantMaxCheckAttempts != got.MaxCheckAttempts {
					t.Errorf("MaxCheckAttempts mismatch; want=%d, got=%d", wantMaxCheckAttempts, got.MaxCheckAttempts)
				}

				wantTemplates := "generic-service"
				if l := len(got.Templates); l != 1 {
					t.Errorf("Templates mismatch; want=%d entries, got=%d", 1, l)
				} else if wantTemplates != got.Templates[0] {
					t.Errorf("Templates mismatch; want=%q, got=%q", wantTemplates, got.Templates[0])
				}

				wantPluginOutputAnnotations := "message"
				if l := len(got.PluginOutputAnnotations); l != 1 {
					t.Errorf("PluginOutputAnnotations mismatch; want=%d entries, got=%d", 1, l)
				} else if wantPluginOutputAnnotations != got.PluginOutputAnnotations[0] {
					t.Errorf("PluginOutputAnnotations mismatch; want=%q, got=%q", wantPluginOutputAnnotations, got.PluginOutputAnnotations[0])
				}

				wantChecksInterval := 12 * time.Hour
				if wantChecksInterval != got.ChecksInterval {
					t.Errorf("ChecksInterval mismatch; want=%q, got=%q", wantChecksInterval, got.ChecksInterval)
				}

				wantKeepFor := 168 * time.Hour
				if wantKeepFor != got.KeepFor {
					t.Errorf("KeepFor mismatch; want=%q, got=%q", wantKeepFor, got.KeepFor)
				}

				wantAlertFingerprintExcludes := "severity"
				if l := len(got.AlertFingerprintExcludes); l != 1 {
					t.Errorf("AlertFingerprintExcludes mismatch; want=%d entries, got=%d", 1, l)
				} else if wantAlertFingerprintExcludes != got.AlertFingerprintExcludes[0] {
					t.Errorf("AlertFingerprintExcludes mismatch; want=%q, got=%q", wantAlertFingerprintExcludes, got.AlertFingerprintExcludes[0])
				}
			},
		},
		"Icinga URLs": {
			haveCLI: append(requiredArgs, "--icinga-url=https://icinga02.localhost:5665/"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantIcingaURL01 := "https://icinga.localhost:5665/"
				wantIcingaURL02 := "https://icinga02.localhost:5665/"

				if l := len(got.IcingaURL); l != 2 {
					t.Fatalf("IcingaURL mismatch; want=%d URLs, got=%d", 2, l)
				}

				if wantIcingaURL01 != got.IcingaURL[0] {
					t.Errorf("IcingaURL mismatch; want=%q, got=%q", wantIcingaURL01, got.IcingaURL[0])
				}

				if wantIcingaURL02 != got.IcingaURL[1] {
					t.Errorf("IcingaURL mismatch; want=%q, got=%q", wantIcingaURL02, got.IcingaURL[1])
				}
			},
		},
		"bogus Icinga URL": {
			haveCLI: append(requiredArgs, "--icinga-url=foo+bar://[1:2:3:4::]:5678/9/10"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantIcingaURL := "foo+bar://[1:2:3:4::]:5678/9/10"

				if l := len(got.IcingaURL); l != 2 {
					t.Fatalf("IcingaURL mismatch; want=%d URLs, got=%d", 2, l)
				}

				if wantIcingaURL != got.IcingaURL[1] {
					t.Errorf("IcingaURL mismatch; want=%q, got=%q", wantIcingaURL, got.IcingaURL[1])
				}
			},
		},
		"log level": {
			haveCLI: append(requiredArgs, "--loglevel=warn"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantLogLevel := LogLevel(slog.LevelWarn)

				if wantLogLevel != got.Loglevel {
					t.Errorf("Loglevel mismatch; want=%v, got=%v", wantLogLevel, got.Loglevel)
				}
			},
		},
		"bogus log level name": {
			haveCLI: append(requiredArgs, "--loglevel=loud"),
			wantErr: `--loglevel: slog: level string "loud": unknown name`,
		},
		"bogus log level number": {
			haveCLI: append(requiredArgs, "--loglevel=info+1"),
			wantErr: `--loglevel: log level must be one of debug, info, warn, or error`,
		},
		"keep for": {
			haveCLI: append(requiredArgs, "--keep-for=3600s"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantKeepFor := 3600 * time.Second

				if wantKeepFor != got.KeepFor {
					t.Errorf("KeepFor mismatch; want=%q, got=%q", wantKeepFor, got.KeepFor)
				}
			},
		},
		"bogus keep for": {
			haveCLI: append(requiredArgs, "--keep-for=forever"),
			wantErr: `--keep-for: expected duration but got "forever": time: invalid duration "forever"`,
		},
		"active checks": {
			haveCLI: append(requiredArgs, "--active-checks"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantActiveChecks := true

				if wantActiveChecks != got.ActiveChecks {
					t.Errorf("ActiveChecks mismatch; want=%t, got=%t", wantActiveChecks, got.ActiveChecks)
				}
			},
		},
		"bogus active checks": {
			haveCLI: append(requiredArgs, "--active-checks=maybe"),
			wantErr: `--active-checks: bool value must be true, 1, yes, false, 0 or no but got "maybe"`,
		},
		"max check attempts": {
			haveCLI: append(requiredArgs, "--max-check-attempts=3"),
			wantCLI: func(t *testing.T, got *CLI) {
				wantMaxCheckAttempts := 3

				if wantMaxCheckAttempts != got.MaxCheckAttempts {
					t.Errorf("MaxCheckAttempts mismatch; want=%d, got=%d", wantMaxCheckAttempts, got.MaxCheckAttempts)
				}
			},
		},
		"bogus max check attempts": {
			haveCLI: append(requiredArgs, "--max-check-attempts=infinite"),
			wantErr: `--max-check-attempts: expected a valid 64 bit int but got "infinite"`,
		},
		"custom severity levels": {
			haveCLI: append(requiredArgs, "--custom-severity-levels=page=3;info=0"),
			wantCLI: func(t *testing.T, got *CLI) {
				if l := len(got.CustomSeverityLevels); l != 2 {
					t.Fatalf("CustomSeverityLevels mismatch; want=%d URLs, got=%d", 2, l)
				}

				if sl, ok := got.CustomSeverityLevels["page"]; !ok {
					t.Errorf("CustomSeverityLevels is missing entry; want=%q", "page")
				} else if sl != StatusCode(3) {
					t.Errorf("CustomSeverityLevels mismatch; want=%d; got=%d", 3, sl)
				}

				if sl, ok := got.CustomSeverityLevels["info"]; !ok {
					t.Errorf("CustomSeverityLevels is missing entry; want=%q", "info")
				} else if sl != StatusCode(0) {
					t.Errorf("CustomSeverityLevels mismatch; want=%d; got=%d", 0, sl)
				}
			},
		},
		"bogus custom severity levels": {
			haveCLI: append(requiredArgs, "--custom-severity-levels=foobar=WARNING"),
			wantErr: `--custom-severity-levels: invalid map value "WARNING"`,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			var subject CLI
			var cli *kong.Kong
			var ctx *kong.Context
			var err error
			cli, err = kong.New(&subject)
			if err != nil {
				t.Fatalf("failed to create commandline parser: %v", err)
			}

			ctx, err = cli.Parse(testCase.haveCLI)
			if err == nil {
				err = ctx.Error
			}

			if err != nil {
				if testCase.wantErr != "" {
					if !strings.Contains(err.Error(), testCase.wantErr) {
						t.Fatalf("expected error to contain %q, got %v", testCase.wantErr, err)
					}
				} else {
					t.Fatalf("commandline parsing yielded an error (%v), wanted none", err)
				}

				return
			} else if testCase.wantErr != "" {
				t.Fatalf("commandline parsing expected an error, got none")
			}

			if testCase.wantCLI != nil {
				testCase.wantCLI(t, &subject)
			}
		})
	}
}
