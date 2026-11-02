// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"encoding/json"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// mapstructure's ",remain" decoder only accepts map fields, so a bare
// interface{} makes Decode fail for any config carrying unknown keys.
func TestMapstructureDecodeKeepsUnknownKeys(t *testing.T) {
	raw := map[string]any{
		"batch":     map[string]any{},
		"extra_key": "value",
	}

	var sp SpanProcessor
	require.NoError(t, mapstructure.Decode(raw, &sp))
	assert.Equal(t, map[string]any{"extra_key": "value"}, sp.AdditionalProperties)

	var cfg OpenTelemetryConfiguration
	require.NoError(t, mapstructure.Decode(map[string]any{
		"file_format": "0.4",
		"extra_key":   "value",
	}, &cfg))
	assert.Equal(t, map[string]any{"extra_key": "value"}, cfg.AdditionalProperties)
}

func TestAdditionalPropertiesNotSerialized(t *testing.T) {
	sp := SpanProcessor{AdditionalProperties: map[string]any{"extra_key": "value"}}

	b, err := json.Marshal(sp)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b))

	y, err := yaml.Marshal(sp)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(y))
}
