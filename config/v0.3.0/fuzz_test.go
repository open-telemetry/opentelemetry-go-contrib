// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzJSON(f *testing.F) {
	b, err := os.ReadFile(filepath.Join("..", "testdata", "v0.3.json"))
	require.NoError(f, err)
	f.Add(b)

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Log("JSON:\n" + string(data))

		var cfg OpenTelemetryConfiguration
		err := json.Unmarshal(b, &cfg)
		if err != nil {
			return
		}

		sdk, err := NewSDK(WithOpenTelemetryConfiguration(cfg))
		if err != nil {
			return
		}

		assert.NoError(t, sdk.Shutdown(context.Background()))
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

		assert.NoError(t, sdk.Shutdown(context.Background()))
	})
}
