// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func FuzzJSON(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("..", "testdata", "v0.3.json"))
	require.NoError(f, err)
	f.Add(b)

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Log("JSON:\n" + string(data))

		var cfg OpenTelemetryConfiguration
		err := json.Unmarshal(data, &cfg)
		if err != nil {
			return
		}

		sdk, err := NewSDK(WithOpenTelemetryConfiguration(cfg))
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		_ = sdk.Shutdown(ctx)
	})
}

func FuzzYAML(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("..", "testdata", "v0.3.yaml"))
	require.NoError(f, err)
	f.Add(b)

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Log("YAML:\n" + string(data))

		cfg, err := ParseYAML(data)
		if err != nil {
			return
		}

		sdk, err := NewSDK(WithOpenTelemetryConfiguration(*cfg))
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		_ = sdk.Shutdown(ctx)
	})
}

func FuzzYAMLWithEnvVars(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("..", "testdata", "v0.3-env-var.yaml"))
	require.NoError(f, err)

	// Add example values for fuzzing - YAML data and all env var values.
	f.Add(b, "false", "4096", "test_string", "true", "42", "3.14", "0xFF", "invalid", "dynamic_key", "replaced_value", "value\\nwith\\tescape")

	f.Fuzz(func(t *testing.T, data []byte, otelSDKDisabled, otelAttrValueLengthLimit, stringValue, boolValue, intValue, floatValue, hexValue, invalidMapValue, envVarInKey, replaceMe, valueWithEscape string) {
		t.Log("YAML with env vars:\n" + string(data))

		// Helper function to check if environment variable value is valid.
		isValidEnvValue := func(value string) bool {
			// Environment variable values cannot contain null bytes.
			return !slices.Contains([]byte(value), 0)
		}

		// Set environment variables used in the test YAML with fuzzed values.
		// Skip if any value contains invalid characters.
		envVars := map[string]string{
			"OTEL_SDK_DISABLED":                 otelSDKDisabled,
			"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT": otelAttrValueLengthLimit,
			"STRING_VALUE":                      stringValue,
			"BOOL_VALUE":                        boolValue,
			"INT_VALUE":                         intValue,
			"FLOAT_VALUE":                       floatValue,
			"HEX_VALUE":                         hexValue,
			"INVALID_MAP_VALUE":                 invalidMapValue,
			"ENV_VAR_IN_KEY":                    envVarInKey,
			"REPLACE_ME":                        replaceMe,
			"VALUE_WITH_ESCAPE":                 valueWithEscape,
		}

		for key, value := range envVars {
			if !isValidEnvValue(value) {
				t.Skipf("Skipping test due to invalid env var value for %s", key)
			}
			t.Setenv(key, value)
		}

		cfg, err := ParseYAML(data)
		if err != nil {
			return
		}

		sdk, err := NewSDK(WithOpenTelemetryConfiguration(*cfg))
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		_ = sdk.Shutdown(ctx)
	})
}
