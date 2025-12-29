// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestInvalidEnvVarName(t *testing.T) {
	_, err := replaceEnvVar("$%&(*&)")
	require.ErrorContains(t, err, errors.New("invalid environment variable name: $%&(*&)").Error())
}

func TestCheckRawConfTypeNil(t *testing.T) {
	err := checkRawConfType([]byte{})
	require.Error(t, err)
	require.ErrorContains(t, err, "unsupported type=<nil> for retrieved config")
}

func TestReplaceEnvVar(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "no environment variables",
			input: "key: value\nother: data",
			want:  "key: value\nother: data",
		},
		{
			name:  "simple environment variable substitution",
			input: "key: ${TEST_VAR}",
			env:   map[string]string{"TEST_VAR": "test_value"},
			want:  "key: test_value",
		},
		{
			name:  "undefined environment variable",
			input: "key: ${UNDEFINED_VAR}",
			want:  "key:",
		},
		{
			name:  "environment variable with default value",
			input: "key: ${UNDEFINED_VAR:-default_value}",
			want:  "key: default_value",
		},
		{
			name:  "environment variable with default value when var is set",
			input: "key: ${DEFINED_VAR:-default_value}",
			env:   map[string]string{"DEFINED_VAR": "actual_value"},
			want:  "key: actual_value",
		},
		{
			name:  "escaped dollar sign - single escape",
			input: "key: $${NOT_REPLACED}",
			want:  "key: ${NOT_REPLACED}",
		},
		{
			name:  "escaped dollar sign - double escape produces single dollar",
			input: "key: $$${TEST_VAR}",
			env:   map[string]string{"TEST_VAR": "test_value"},
			want:  "key: $test_value",
		},
		{
			name:  "escaped dollar sign - triple escape",
			input: "key: $$$${NOT_REPLACED}",
			want:  "key: $${NOT_REPLACED}",
		},
		{
			name:  "mixed escaped and unescaped",
			input: "key1: ${REPLACE_ME}\nkey2: $${NOT_REPLACED}",
			env:   map[string]string{"REPLACE_ME": "replaced"},
			want:  "key1: replaced\nkey2: ${NOT_REPLACED}",
		},
		{
			name:  "environment variable in key position",
			input: "${KEY_VAR}: value",
			env:   map[string]string{"KEY_VAR": "dynamic_key"},
			want:  "${KEY_VAR}: value",
		},
		{
			name:  "multiple environment variables in same line",
			input: "key: ${VAR1} and ${VAR2}",
			env: map[string]string{
				"VAR1": "first",
				"VAR2": "second",
			},
			want: "key: first and second",
		},
		{
			name:  "environment variable with spaces in default",
			input: "key: ${UNDEFINED:-default with spaces}",
			want:  "key: default with spaces",
		},
		{
			name:  "nested env vars in default are treated literally",
			input: "key: ${UNDEFINED:-${FALLBACK_VAR}}",
			env:   map[string]string{"FALLBACK_VAR": "fallback_value"},
			want:  "key: ${FALLBACK_VAR}",
		},
		{
			name:  "boolean environment variable",
			input: "enabled: ${BOOL_VAR}",
			env:   map[string]string{"BOOL_VAR": "true"},
			want:  "enabled: true",
		},
		{
			name:  "numeric environment variable",
			input: "count: ${NUM_VAR}",
			env:   map[string]string{"NUM_VAR": "42"},
			want:  "count: 42",
		},
		{
			name:  "hex environment variable",
			input: "value: ${HEX_VAR}",
			env:   map[string]string{"HEX_VAR": "0xFF"},
			want:  "value: 0xFF",
		},
		{
			name:  "alternative env syntax",
			input: "key: ${env:TEST_VAR}",
			env:   map[string]string{"TEST_VAR": "env_value"},
			want:  "key: env_value",
		},
		{
			name:  "quoted environment variable",
			input: "key: \"${QUOTED_VAR}\"",
			env:   map[string]string{"QUOTED_VAR": "quoted_value"},
			want:  "key: \"quoted_value\"",
		},
		{
			name:  "environment variable with special characters",
			input: "key: ${SPECIAL_VAR}",
			env:   map[string]string{"SPECIAL_VAR": "value\\nwith\\tescape"},
			want:  "key: value\\nwith\\tescape",
		},
		{
			name:  "escape sequence in regular text",
			input: "key: a $$ b",
			want:  "key: a $ b",
		},
		{
			name:  "no escape sequence with single dollar",
			input: "key: a $ b",
			want:  "key: a $ b",
		},
		{
			name: "complex YAML with multiple substitutions",
			input: `service:
    name: ${SERVICE_NAME:-default-service}
    version: ${SERVICE_VERSION}
config:
    endpoint: ${ENDPOINT}
    escaped: $${NOT_REPLACED}`,
			env: map[string]string{
				"SERVICE_VERSION": "1.0.0",
				"ENDPOINT":        "http://localhost:8080",
			},
			want: `service:
    name: default-service
    version: 1.0.0
config:
    endpoint: http://localhost:8080
    escaped: ${NOT_REPLACED}`,
		},
		{
			name:    "YAML injection causes error",
			input:   "key: ${MALICIOUS_VAR}",
			env:     map[string]string{"MALICIOUS_VAR": "value\\nkey2: injected"},
			wantErr: true,
		},
		{
			name:    "error case - invalid YAML type",
			input:   "key: ${INVALID_TYPE_VAR}",
			env:     map[string]string{"INVALID_TYPE_VAR": "!!int NaN"},
			wantErr: true,
		},
		{
			name:    "error case - invalid substitution syntax",
			input:   "key: ${ERR_INVALID_SUFFIX:?error}",
			env:     map[string]string{"ERR_INVALID_SUFFIX": "something"},
			wantErr: true,
		},
		{
			name:  "pipe test",
			input: "key: ${PIPE_VAR}",
			env:   map[string]string{"PIPE_VAR": "value|with$|pipes"},
			want:  "key: value|with$|pipes",
		},
		{
			name:  "$$ escape sequence is replaced with $",
			input: "key: $${STRING_VALUE:-${STRING_VALUE}}",
			env:   map[string]string{"STRING_VALUE": "value"},
			want:  "key: ${STRING_VALUE:-value}",
		},
		{
			name:  "undefined key with escape sequence in fallback",
			input: "key: ${UNDEFINED_KEY:-$${UNDEFINED_KEY}}",
			want:  "key: ${UNDEFINED_KEY}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := ReplaceEnvVars([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, string(got))
		})
	}
}

// Tests of https://opentelemetry.io/docs/specs/otel/configuration/data-model/ with some corrections.
func TestSpecExamples(t *testing.T) {
	t.Setenv("STRING_VALUE", "value")
	t.Setenv("BOOL_VALUE", "true")
	t.Setenv("INT_VALUE", "1")
	t.Setenv("FLOAT_VALUE", "1.1")
	t.Setenv("HEX_VALUE", "0xdeadbeef")                   // A valid integer value (i.e. 3735928559) written in hexadecimal
	t.Setenv("INVALID_MAP_VALUE", "value\nkey:value")     // An invalid attempt to inject a map key into the YAML
	t.Setenv("DO_NOT_REPLACE_ME", "Never use this value") // An unused environment variable
	t.Setenv("REPLACE_ME", "${DO_NOT_REPLACE_ME}")        // A valid replacement text, used verbatim, not replaced with "Never use this value"
	t.Setenv("VALUE_WITH_ESCAPE", "value$$")

	tests := []struct {
		yamlInput  string
		yamlOutput string
		tagURI     string
		notes      string
		wantErr    bool
	}{
		{ // 0
			yamlInput:  "key: ${STRING_VALUE}",
			yamlOutput: "key: value",
			tagURI:     "tag:yaml.org,2002:str",
			notes:      "YAML parser resolves to string",
		},
		{ // 1
			yamlInput:  "key: ${BOOL_VALUE}",
			yamlOutput: "key: true",
			tagURI:     "tag:yaml.org,2002:bool",
			notes:      "YAML parser resolves to true",
		},
		{ // 2
			yamlInput:  "key: ${INT_VALUE}",
			yamlOutput: "key: 1",
			tagURI:     "tag:yaml.org,2002:int",
			notes:      "YAML parser resolves to int",
		},
		{ // 3
			yamlInput:  "key: ${FLOAT_VALUE}",
			yamlOutput: "key: 1.1",
			tagURI:     "tag:yaml.org,2002:float",
			notes:      "YAML parser resolves to float",
		},
		{ // 4
			yamlInput:  "key: ${HEX_VALUE}",
			yamlOutput: "key: 0xdeadbeef",
			tagURI:     "tag:yaml.org,2002:int",
			notes:      "YAML parser resolves to int 3735928559",
		},
		{ // 5
			yamlInput:  `key: "${STRING_VALUE}"`,
			yamlOutput: `key: "value"`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Double quoted to force coercion to string "value"`,
		},
		{ // 6
			yamlInput:  `key: "${BOOL_VALUE}"`,
			yamlOutput: `key: "true"`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Double quoted to force coercion to string "true"`,
		},
		{ // 7
			yamlInput:  `key: "${INT_VALUE}"`,
			yamlOutput: `key: "1"`,
			tagURI:     `tag:yaml.org,2002:str`,
			notes:      `Double quoted to force coercion to string "1"`,
		},
		{ // 8
			yamlInput:  `key: "${FLOAT_VALUE}"`,
			yamlOutput: `key: "1.1"`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Double quoted to force coercion to string "1.1"`,
		},
		{ // 9
			yamlInput:  `key: "${HEX_VALUE}"`,
			yamlOutput: `key: "0xdeadbeef"`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Double quoted to force coercion to string "0xdeadbeef"`,
		},
		{ // 10
			yamlInput:  "key: ${env:STRING_VALUE}",
			yamlOutput: `key: value`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      "Alternative env: syntax",
		},
		{ // 11
			yamlInput:  "key: ${INVALID_MAP_VALUE}",
			yamlOutput: "key: |-\n    value\n    key:value",
			tagURI:     "tag:yaml.org,2002:str",
			notes:      "Map structure resolves to string and not expanded",
		},
		{ // 12
			yamlInput:  "key: foo ${STRING_VALUE} ${FLOAT_VALUE}",
			yamlOutput: `key: foo value 1.1`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Multiple references are injected and resolved to string`,
		},
		{ // 13
			yamlInput:  "key: ${UNDEFINED_KEY}",
			yamlOutput: `key:`,
			tagURI:     "tag:yaml.org,2002:null",
			notes:      `Undefined env var is replaced with "" and resolves to null`,
		},
		{ // 14
			yamlInput:  "key: ${UNDEFINED_KEY:-fallback}",
			yamlOutput: `key: fallback`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Undefined env var results in substitution of default value fallback`,
		},
		{ // 15
			yamlInput:  "${STRING_VALUE}: value",
			yamlOutput: `${STRING_VALUE}: value`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Usage of substitution syntax in keys is ignored`,
		},
		{ // 16
			yamlInput:  "key: ${REPLACE_ME}",
			yamlOutput: `key: ${DO_NOT_REPLACE_ME}`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Value of env var REPLACE_ME is ${DO_NOT_REPLACE_ME}, and is not substituted recursively`,
		},
		{ // 17
			yamlInput:  "key: ${UNDEFINED_KEY:-${STRING_VALUE}}",
			yamlOutput: `key: ${STRING_VALUE}`, // added missing key:
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Undefined env var results in substitution of default value ${STRING_VALUE}, and is not substituted recursively`,
		},
		{ // 18
			yamlInput:  "key: ${STRING_VALUE:?error}",
			yamlOutput: "",
			tagURI:     "",
			notes:      `Invalid substitution reference produces parse error`,
			wantErr:    true,
		},
		{ // 19
			yamlInput:  "key: $${STRING_VALUE}",
			yamlOutput: `key: ${STRING_VALUE}`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, {STRING_VALUE} does not match substitution syntax`,
		},
		{ // 20
			yamlInput:  "key: $$${STRING_VALUE}",
			yamlOutput: `key: $value`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, ${STRING_VALUE} is replaced with value`,
		},
		{ // 21
			yamlInput:  "key: $$$${STRING_VALUE}",
			yamlOutput: `key: $${STRING_VALUE}`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, $$ escape sequence is replaced with $, {STRING_VALUE} does not match substitution syntax`,
		},
		{ // 22
			yamlInput:  "key: $${STRING_VALUE:-fallback}",
			yamlOutput: `key: ${STRING_VALUE:-fallback}`, // added missing key:
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, {STRING_VALUE:-fallback} does not match substitution syntax`,
		},
		{ // 23
			yamlInput:  "key: $${STRING_VALUE:-${STRING_VALUE}}",
			yamlOutput: `key: ${STRING_VALUE:-value}`, // added missing key:
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, leaving {STRING_VALUE:-${STRING_VALUE}}, ${STRING_VALUE} is replaced with value`,
		},
		{ // 24
			yamlInput: "key: ${UNDEFINED_KEY:-$${UNDEFINED_KEY}}",
			// yamlOutput: `key: ${UNDEFINED_KEY:-${UNDEFINED_KEY}}`,
			// this is what the spec tells, but I object,
			// 1. we read until :-
			// 2. encounter $$ and substitute with $
			// 3. then reading {UNDEFINED_KEY
			// 4. encounter }, that will conclude the env substitution, thus we have ${UNDEFINED_KEY:-${UNDEFINED_KEY}
			//    that is perfectly valid and will result int ${UNDEFINED_KEY
			// 5. encounter an unrelated }, that is just printed
			yamlOutput: `key: ${UNDEFINED_KEY}`, // added missing key:
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $, leaving ${UNDEFINED_KEY:- before and ${UNDEFINED_KEY}} after which do not match substitution syntax`,
		},
		{ // 25
			yamlInput:  "key: ${VALUE_WITH_ESCAPE}",
			yamlOutput: `key: value$$`, // added missing key:
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `Value of env var VALUE_WITH_ESCAPE is value$$, which is substituted without escaping`,
		},
		{ // 26
			yamlInput:  "key: a $$ b",
			yamlOutput: `key: a $ b`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `$$ escape sequence is replaced with $`,
		},
		{ // 27
			yamlInput:  "key: a $ b",
			yamlOutput: `key: a $ b`,
			tagURI:     "tag:yaml.org,2002:str",
			notes:      `No escape sequence, no substitution references, value is left unchanged`,
		},
	}

	for idx, tt := range tests {
		t.Run(fmt.Sprintf("SpecExample-%d", idx), func(t *testing.T) {
			got, err := ReplaceEnvVars([]byte(tt.yamlInput))

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.yamlOutput, string(got), tt.notes)

			if tt.wantErr {
				return
			}

			// checking the resulting type
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal(got, &node))

			require.True(t,
				len(node.Content) == 1 && len(node.Content[0].Content) == 2,
				"type check node wrong structure")

			// the YAML library tags are with !! instead of the long version
			wantType := strings.ReplaceAll(tt.tagURI, "tag:yaml.org,2002:", "!!")

			assert.Equal(t, wantType, node.Content[0].Content[1].Tag)
		})
	}
}
