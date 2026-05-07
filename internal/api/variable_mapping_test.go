// Licensed under "BSD 3-Clause". See LICENSE file.

package api

import (
	"io"
	"log/slog"
	"testing"

	"github.com/NETWAYS/alertmanager-icinga-bridge/internal/icinga2"

	"github.com/stretchr/testify/assert"
)

var mapIcingaVariableTest = map[string]struct {
	iK  string
	iV  string
	oK  string
	oV  interface{}
	err error
}{
	"not mapped":    {"foo", "bar", "foo", "bar", ErrorNotAMappingKey},
	"mapped number": {"icinga_number_foo", "42", "foo", 42, nil},
	"mapped string": {"icinga_string_foo", "bar", "foo", "bar", nil},
	"unknown":       {"icinga_unknown_foo", "bar", "", nil, ErrorUnknownMappingType},
}

func TestMapIcingaVariable(t *testing.T) {
	for name, test := range mapIcingaVariableTest {
		t.Run(name, func(t *testing.T) {
			k, v, err := mapIcingaVariable(test.iK, test.iV)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.oK, k)
			assert.Equal(t, test.oV, v)
		})
	}
}

func TestMapIcingaVariables(t *testing.T) {
	vars := make(icinga2.Vars)
	kv := map[string]string{
		"a":                "a",
		"icinga_number_b":  "42",
		"icinga_string_c":  "c",
		"icinga_unknown_d": "d",
		"icinga_number_e":  "e",
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	vars = mapIcingaVariables(vars, kv, "pre_", logger)
	assert.Equal(t, icinga2.Vars{
		"pre_a":                "a",
		"pre_icinga_number_b":  "42",
		"pre_icinga_string_c":  "c",
		"pre_icinga_unknown_d": "d",
		"pre_icinga_number_e":  "e",
		"b":                    42,
		"c":                    "c",
	}, vars)
}

func TestAddStaticVariables(t *testing.T) {
	vars := make(icinga2.Vars)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	staticVars := map[string]string{
		"a": "a",
		"b": "b",
	}
	vars = addStaticIcingaVariables(vars, staticVars, logger)
	assert.Equal(t, icinga2.Vars{
		"a": "a",
		"b": "b",
	}, vars)
}

func TestAddStaticVariablesNoOverwrite(t *testing.T) {
	vars := make(icinga2.Vars)
	vars["a"] = "z"
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	staticVars := map[string]string{
		"a": "a",
		"b": "b",
	}
	vars = addStaticIcingaVariables(vars, staticVars, logger)
	assert.Equal(t, icinga2.Vars{
		"a": "z",
		"b": "b",
	}, vars)
}
