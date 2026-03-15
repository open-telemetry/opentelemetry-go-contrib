// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			want:  "key: ",
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
			want:  "dynamic_key: value",
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
