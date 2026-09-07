// SPDX-License-Identifier: Apache-2.0

package icinga2

import (
	"encoding/json"
	"strconv"
	"strings"
)

type PerfData []string

type Command []string

type TimeStamp string

type Vars map[string]any

type QueryFilter struct {
	Filter string `json:"filter"`
}

type ExitStatus int

const (
	ExitStatusOK = iota
	ExitStatusWarning
	ExitStatusCritical
	ExitStatusUnknown
)

// ParseExitStatus attempts to parse the given value
// as number and falls back to matching against
// well-known names. if either of those fails,
// an error is returned.
func ParseExitStatus(s string) (ExitStatus, error) {
	i, err := strconv.ParseInt(s, 10, 8)
	if err == nil {
		return ExitStatus(i), nil
	}

	switch v := strings.ToUpper(s); v {
	case "OK":
		return ExitStatusOK, nil
	case "WARNING":
		return ExitStatusWarning, nil
	case "CRITICAL":
		return ExitStatusCritical, nil
	case "UNKNOWN":
		return ExitStatusUnknown, nil
	}

	// return the numeric parsing error from above
	return ExitStatusOK, err
}

// Validate determines if the instance is a well-known
// implementation
func (es ExitStatus) Validate() bool {
	return ExitStatusOK <= es && es <= ExitStatusUnknown
}

// ExitCode returns the process exit code this instance represents.
func (es ExitStatus) ExitCode() int {
	return int(es)
}

// String returns the name for the exit code.
// values outside of the valid range are their
// numeric representation prefixed with "EXIT_"
func (es ExitStatus) String() string {
	switch es {
	case ExitStatusOK:
		return "OK"
	case ExitStatusWarning:
		return "WARNING"
	case ExitStatusCritical:
		return "CRITICAL"
	case ExitStatusUnknown:
		return "UNKNOWN"
	default:
		return "EXIT_" + strconv.FormatInt(int64(es), 10)
	}
}

// MarshalJSON implements [encoding/json.Marshaler] by calling
// [ExitStatus.MarshalText], as the preferred way of serializing exit codes is
// using their numeric representation.
func (es ExitStatus) MarshalJSON() ([]byte, error) {
	return es.MarshalText()
}

// MarshalText implements [encoding.TextMarshaler] by calling
// [strconv.FormatInt] on itself.
func (es ExitStatus) MarshalText() ([]byte, error) {
	s := strconv.FormatInt(int64(es), 10)

	return []byte(s), nil
}

type Action struct {
	ExitStatus      ExitStatus `json:"exit_status"`
	PluginOutput    string     `json:"plugin_output"`
	PerformanceData PerfData   `json:"performance_data,omitempty"`
	CheckCommand    Command    `json:"check_command,omitempty"`
	CheckSource     string     `json:"check_source,omitempty"`
	ExecutionStart  TimeStamp  `json:"execution_start,omitempty"`
	ExecutionEnd    TimeStamp  `json:"execution_end,omitempty"`
	TTL             int        `json:"ttl"`
	Filter          string     `json:"filter"`
	Type            string     `json:"type"`
}

type Host struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Zone        string `json:"zone,omitempty"`
}

type HostResults struct {
	Results []struct {
		Host Host `json:"attrs"`
	} `json:"results"`
}

type Service struct {
	Name               string   `json:"name,omitempty"`
	DisplayName        string   `json:"display_name"`
	HostName           string   `json:"host_name"`
	CheckCommand       string   `json:"check_command"`
	Notes              string   `json:"notes"`
	NotesURL           string   `json:"notes_url"`
	ActionURL          string   `json:"action_url"`
	CheckPeriod        string   `json:"check_period,omitempty"`
	Zone               string   `json:"zone,omitempty"`
	EnableActiveChecks bool     `json:"enable_active_checks"`
	DowntimeDepth      int      `json:"downtime_depth,omitempty"`
	CheckInterval      float64  `json:"check_interval"`
	RetryInterval      float64  `json:"retry_interval"`
	MaxCheckAttempts   float64  `json:"max_check_attempts"`
	State              float64  `json:"state,omitempty"`
	LastStateChange    float64  `json:"last_state_change,omitempty"`
	Templates          []string `json:"templates,omitempty"`
	Vars               Vars     `json:"vars,omitempty"`
}

func (s *Service) FullName() string {
	return s.HostName + "!" + s.Name
}

func (s *Service) HasDowntime() bool {
	return s.DowntimeDepth > 0
}

func (s Service) MarshalJSON() ([]byte, error) {
	// Prevent json.Marshal() recursion
	type service Service

	svc := service(s)

	// Clear top-level Vars field, so it's not added to the marshalled JSON. We marshal all service variables into individual top-level `vars.<variable name>` fields below.
	svc.Vars = Vars{}

	serviceAsJSON, err := json.Marshal(svc)
	if err != nil {
		return nil, err
	}

	var serviceAsMap map[string]any

	errUnmarshal := json.Unmarshal(serviceAsJSON, &serviceAsMap)

	if errUnmarshal != nil {
		return nil, errUnmarshal
	}

	// This loop flattens the json, so each var will be at the same level
	for k, v := range flatten(s.Vars) {
		serviceAsMap["vars."+k] = v
	}

	return json.Marshal(serviceAsMap)
}

type ServiceResults struct {
	Results []struct {
		Service Service `json:"attrs"`
	} `json:"results"`
}

type ServiceCreate struct {
	Templates []string `json:"templates"`
	Attrs     Service  `json:"attrs"`
}

func flatten(m map[string]any) map[string]any {
	flat := map[string]any{}

	for k, v := range m {
		switch child := v.(type) {
		case map[string]any:
			flatChild := flatten(child)
			for ck, cv := range flatChild {
				flat[k+"."+ck] = cv
			}
		case []any:
			for i := range child {
				flat[k+"["+strconv.Itoa(i)+"]"] = child[i]
			}
		default:
			flat[k] = v
		}
	}

	return flat
}
