// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/internal/provider"

import (
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
		},
		{
			name:       "invalid variable type list",
			uri:        "LIST_VALUE",
			envVarName: "LIST_VALUE",
			val:        `["one", "two"]`,
			wantValue:  []byte(nil),
		},
		{
			name:       "handle invalid yaml",
			uri:        "NOT_A_NUMBER_VALUE",
			envVarName: "NOT_A_NUMBER_VALUE",
			val:        `!!int NaN`,
			wantValue:  []byte(nil),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.envVarName) > 0 {
				t.Setenv(tt.envVarName, tt.val)
			}
			got := ReplaceEnvVar(tt.uri)
			require.Equal(t, tt.wantValue, got)
		})
	}
}
