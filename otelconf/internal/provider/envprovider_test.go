// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"errors"
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
			name:  "environment variable in key position is not substituted",
			input: "${KEY_VAR}: value",
			env:   map[string]string{"KEY_VAR": "dynamic_key"},
			want:  "${KEY_VAR}: value",
		},
		{
			name:  "environment variables in a sequence",
			input: "key:\n  - ${VAR1}\n  - ${VAR2}\n",
			env:   map[string]string{"VAR1": "first", "VAR2": "2"},
			want:  "key:\n  - first\n  - 2\n",
		},
		{
			name:  "aliased environment variable is substituted once",
			input: "key: &anchor $$${TEST_VAR}\nkey2: *anchor\n",
			env:   map[string]string{"TEST_VAR": "test_value"},
			want:  "key: $test_value\nkey2: $test_value\n",
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
			name:  "quoted boolean environment variable is a string",
			input: "key: \"${BOOL_VAR}\"",
			env:   map[string]string{"BOOL_VAR": "true"},
			want:  "key: \"true\"",
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
			input: "service:\n" +
				"  name: ${SERVICE_NAME:-default-service}\n" +
				"  version: ${SERVICE_VERSION}\n" +
				"config:\n" +
				"  endpoint: ${ENDPOINT}\n" +
				"  escaped: $${NOT_REPLACED}\n",
			env: map[string]string{
				"SERVICE_VERSION": "1.0.0",
				"ENDPOINT":        "http://localhost:8080",
			},
			want: "service:\n" +
				"  name: default-service\n" +
				"  version: 1.0.0\n" +
				"config:\n" +
				"  endpoint: http://localhost:8080\n" +
				"  escaped: ${NOT_REPLACED}\n",
		},
		{
			name:    "YAML injection causes error",
			input:   "key: ${MALICIOUS_VAR}",
			env:     map[string]string{"MALICIOUS_VAR": "value\\nkey2: injected"},
			wantErr: true,
		},
		{
			name:  "block scalar value does not add keys",
			input: "parent:\n  key: ${MALICIOUS_VAR}\n",
			env:   map[string]string{"MALICIOUS_VAR": "|\nkey2: injected"},
			want:  "parent:\n  key: \"\"\n",
		},
		{
			name:  "document marker in value does not truncate the document",
			input: "key: ${MALICIOUS_VAR}\nkey2: kept\n",
			env:   map[string]string{"MALICIOUS_VAR": "value\n---\n"},
			want:  "key: value\nkey2: kept\n",
		},
		{
			name:  "value cannot close a double quoted scalar",
			input: `key: "${MALICIOUS_VAR}"`,
			env:   map[string]string{"MALICIOUS_VAR": "\"\nkey2: injected\nkey3: \""},
			want:  `key: "\"\nkey2: injected\nkey3: \""`,
		},
		{
			name:  "value cannot close a single quoted scalar",
			input: "key: '${MALICIOUS_VAR}'",
			env:   map[string]string{"MALICIOUS_VAR": "'\nkey2: injected\nkey3: '"},
			want:  `key: "'\nkey2: injected\nkey3: '"`,
		},
		{
			name:  "trailing line break in a whole value",
			input: "key: ${TRAILING_NEWLINE_VAR}",
			env:   map[string]string{"TRAILING_NEWLINE_VAR": "value\r\n"},
			want:  "key: value",
		},
		{
			name:  "trailing line break before a literal suffix",
			input: "parent:\n  key: ${TRAILING_NEWLINE_VAR}---\n  key2: kept\n",
			env:   map[string]string{"TRAILING_NEWLINE_VAR": "value\r\n"},
			want:  "parent:\n  key: \"value\\r\\n---\"\n  key2: kept\n",
		},
		{
			name:    "error case - invalid value followed by a valid reference",
			input:   "key: ${MALICIOUS_VAR}\nkey2: ${TEST_VAR}",
			env:     map[string]string{"MALICIOUS_VAR": "key3: injected", "TEST_VAR": "value"},
			wantErr: true,
		},
		{
			name:    "error case - invalid substitution syntax followed by a valid reference",
			input:   "key: ${ERR_INVALID_SUFFIX:?error}\nkey2: ${TEST_VAR}",
			env:     map[string]string{"TEST_VAR": "value"},
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
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)

			// Compare decoded values so resolved types are checked too.
			var gotVal, wantVal any
			require.NoError(t, got.Decode(&gotVal))
			require.NoError(t, yaml.Unmarshal([]byte(tt.want), &wantVal))
			assert.Equal(t, wantVal, gotVal)
		})
	}
}

func FuzzReplaceEnvVars(f *testing.F) {
	f.Add("simple")
	f.Add("|\nkey2: injected")
	f.Add("value\n---\n")
	f.Add("\"\nkey2: injected\nkey3: \"")
	f.Add("'\nkey2: injected\nkey3: '")
	f.Add("value\r\n")

	templates := []string{
		"parent:\n  target: ${V}\n  sentinel: keep\nafter: keep\n",
		"parent:\n  target: \"${V}\"\n  sentinel: keep\nafter: keep\n",
		"parent:\n  target: '${V}'\n  sentinel: keep\nafter: keep\n",
		"parent:\n  target: prefix-${V}-suffix\n  sentinel: keep\nafter: keep\n",
		"parent:\n  target: ${V}---\n  sentinel: keep\nafter: keep\n",
	}

	f.Fuzz(func(t *testing.T, val string) {
		if strings.ContainsRune(val, '\x00') {
			t.Skip()
		}
		t.Setenv("V", val)
		for _, tmpl := range templates {
			node, err := ReplaceEnvVars([]byte(tmpl))
			if err != nil {
				continue
			}
			var got map[string]any
			if err := node.Decode(&got); err != nil {
				continue
			}
			require.Equal(t, "keep", got["after"])
			require.Len(t, got, 2)
			parent, ok := got["parent"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "keep", parent["sentinel"])
			require.Len(t, parent, 2)
			require.IsNotType(t, map[string]any(nil), parent["target"])
			require.IsNotType(t, []any(nil), parent["target"])
		}
	})
}
