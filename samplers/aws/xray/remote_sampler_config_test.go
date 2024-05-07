// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

// assert that user provided values are tied to config.
func TestNewConfig(t *testing.T) {
	endpoint, err := url.Parse("https://127.0.0.1:5000")
	require.NoError(t, err)

	cfg, err := newConfig(WithSamplingRulesPollingInterval(400*time.Second), WithEndpoint(*endpoint), WithLogger(logr.Logger{}))
	require.NoError(t, err)

	assert.Equal(t, cfg.samplingRulesPollingInterval, 400*time.Second)
	assert.Equal(t, cfg.endpoint, *endpoint)
	assert.Equal(t, cfg.logger, logr.Logger{})
}

// assert that when user did not provide values are then config would be picked up from default values.
func TestDefaultConfig(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:2000")
	require.NoError(t, err)

	cfg, err := newConfig()
	require.NoError(t, err)

	assert.Equal(t, cfg.samplingRulesPollingInterval, 300*time.Second)
	assert.Equal(t, cfg.endpoint, *endpoint)
	assert.Equal(t, cfg.logger, defaultLogger)
}

// assert when some config is provided by user then other config will be picked up from default config.
func TestPartialUserProvidedConfig(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:2000")
	require.NoError(t, err)

	cfg, err := newConfig(WithSamplingRulesPollingInterval(500 * time.Second))
	require.NoError(t, err)

	assert.Equal(t, cfg.samplingRulesPollingInterval, 500*time.Second)
	assert.Equal(t, cfg.endpoint, *endpoint)
	assert.Equal(t, cfg.logger, defaultLogger)
}

// assert that valid endpoint would not result in an error.
func TestValidEndpoint(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:2000")
	require.NoError(t, err)

	cfg, err := newConfig(WithEndpoint(*endpoint))
	require.NoError(t, err)

	assert.Equal(t, cfg.endpoint, *endpoint)
}

// assert that host name with special character would not result in an error.
func TestValidateHostNameWithSpecialCharacterEndpoint(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1@:2000")
	require.NoError(t, err)

	cfg, err := newConfig(WithEndpoint(*endpoint))
	require.NoError(t, err)

	assert.Equal(t, cfg.endpoint, *endpoint)
}

// assert that endpoint without host name would not result in an error.
func TestValidateInvalidEndpoint(t *testing.T) {
	endpoint, err := url.Parse("https://")
	require.NoError(t, err)

	cfg, err := newConfig(WithEndpoint(*endpoint))
	require.NoError(t, err)

	assert.Equal(t, cfg.endpoint, *endpoint)
}

// assert negative sampling rules interval leads to an error.
func TestValidateConfigNegativeDuration(t *testing.T) {
	_, err := newConfig(WithSamplingRulesPollingInterval(-300 * time.Second))
	assert.Error(t, err)
}

// assert positive sampling rules interval.
func TestValidateConfigPositiveDuration(t *testing.T) {
	_, err := newConfig(WithSamplingRulesPollingInterval(300 * time.Second))
	assert.NoError(t, err)
}
