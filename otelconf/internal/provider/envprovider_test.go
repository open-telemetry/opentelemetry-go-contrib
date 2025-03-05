// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/internal/provider"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceEnvVar(t *testing.T) {
	for _, tt := range []struct {
		name       string
		envVarName string
		uri        string
		val        string
		wantValue  any
		wantErr    error
	}{
		{
			name:       "int value",
			envVarName: "INT_VALUE",
			uri:        "INT_VALUE",
			val:        "1",
			wantValue:  []byte("1"),
		},
		{
			name:       "string value",
			envVarName: "STRING_VALUE",
			uri:        "STRING_VALUE",
			val:        "this is a string",
			wantValue:  []byte("this is a string"),
		},
		{
			name:      "invalid env var name",
			uri:       "$%&(*&)",
			wantValue: []byte(nil),
			wantErr:   errors.New("invalid environment variable name: $%&(*&)"),
		},
		{
			name:      "unset value has default",
			uri:       "THIS_VALUE_IS_NOT_SET:-this is a default value",
			wantValue: []byte("this is a default value"),
		},
		{
			name:      "unset value no default",
			uri:       "THIS_VALUE_IS_NOT_SET",
			wantValue: []byte(nil),
		},
		{
			name:       "invalid variable type map",
			uri:        "MAP_VALUE",
			envVarName: "MAP_VALUE",
			val: `key:
  value: something`,
			wantValue: []byte(nil),
			wantErr:   errors.New("invalid value type: unsupported type=map[string]interface {} for retrieved config, ensure that values are wrapped in quotes"),
		},
		{
			name:       "invalid variable type list",
			uri:        "LIST_VALUE",
			envVarName: "LIST_VALUE",
			val:        `["one", "two"]`,
			wantValue:  []byte(nil),
			wantErr:    errors.New("invalid value type: unsupported type=[]interface {} for retrieved config, ensure that values are wrapped in quotes"),
		},
		{
			name:       "handle invalid yaml",
			uri:        "NOT_A_NUMBER_VALUE",
			envVarName: "NOT_A_NUMBER_VALUE",
			val:        `!!int NaN`,
			wantValue:  []byte(nil),
			wantErr:    errors.New("invalid value type: yaml: cannot decode !!str `NaN` as a !!int"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.envVarName) > 0 {
				t.Setenv(tt.envVarName, tt.val)
			}
			got, err := replaceEnvVar(tt.uri)
			require.Equal(t, tt.wantValue, got)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}

		})
	}
}

func TestReplaceEnvVars(t *testing.T) {
	for _, tt := range []struct {
		name     string
		in       string
		setupEnv func(t *testing.T)
		want     string
		wantErr  error
	}{
		{
			name: "no replace",
			in:   "data",
			want: "data",
		},
		{
			name:     "env var present",
			in:       "data: ${TEST_ENV_VAR}",
			setupEnv: func(t *testing.T) { t.Setenv("TEST_ENV_VAR", "value") },
			want:     "data: value",
		},
		{
			name: "env var missing, default present",
			in:   "data: ${TEST_ENV_VAR:-\"val\"}",
			want: "data: \"val\"",
		},
		{
			name: "unset environment variable config",
			in:   "data: ${TEST_ENV_VAR}",
			want: "data: ",
		},
		{
			name:     "handle invalid yaml",
			in:       "data: ${TEST_NOT_A_NUMBER_VALUE}",
			setupEnv: func(t *testing.T) { t.Setenv("TEST_NOT_A_NUMBER_VALUE", "!!int NaN") },
			wantErr:  errors.New("invalid value type: yaml: cannot decode !!str `NaN` as a !!int"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv(t)
			}

			got, err := ReplaceEnvVars([]byte(tt.in))
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, []byte(tt.want), got)
			}
		})
	}
}
