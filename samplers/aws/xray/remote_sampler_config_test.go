// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/stretchr/testify/assert"
)

// assert that user provided values are tied to config.
func TestNewConfig(t *testing.T) {
	cfg := newConfig(WithSamplingRulesPollingInterval(400*time.Second), WithEndpoint("127.0.0.1:5000"), WithLogger(logr.Logger{}))

	assert.Equal(t, cfg.samplingRulesPollingInterval, 400*time.Second)
	assert.Equal(t, cfg.endpoint, "127.0.0.1:5000")
	assert.Equal(t, cfg.logger, logr.Logger{})
}

// assert that when user did not provide values are then config would be picked up from default values.
func TestDefaultConfig(t *testing.T) {
	cfg := newConfig()

	assert.Equal(t, cfg.samplingRulesPollingInterval, 300*time.Second)
	assert.Equal(t, cfg.endpoint, "127.0.0.1:2000")
	assert.Equal(t, cfg.logger, stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error}))
}

// assert when some config is provided by user then other config will be picked up from default config.
func TestPartialUserProvidedConfig(t *testing.T) {
	cfg := newConfig(WithSamplingRulesPollingInterval(500 * time.Second))

	assert.Equal(t, cfg.samplingRulesPollingInterval, 500*time.Second)
	assert.Equal(t, cfg.endpoint, "127.0.0.1:2000")
	assert.Equal(t, cfg.logger, stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error}))
}

// assert not expected endpoint format leads to an error (expected format: "127.0.0.1:2020").
func TestValidateConfigIncorrectEndpoint(t *testing.T) {
	cfg := newConfig(WithEndpoint("http://127.0.0.1:2000"))

	err := validateConfig(cfg)
	assert.Error(t, err)
}

// assert not expected endpoint format leads to an error (expected format: "127.0.0.1:2020").
func TestValidateConfigSpecialCharacterEndpoint(t *testing.T) {
	cfg := newConfig(WithEndpoint("@127.0.0.1:2000"))

	err := validateConfig(cfg)
	assert.Error(t, err)
}

// assert not expected endpoint format leads to an error (expected format: "127.0.0.1:2020").
func TestValidateConfigLocalHost(t *testing.T) {
	cfg := newConfig(WithEndpoint("localhost:2000"))

	err := validateConfig(cfg)
	assert.NoError(t, err)
}

// assert not expected endpoint format leads to an error (expected format: "127.0.0.1:2020").
func TestValidateConfigInvalidPort(t *testing.T) {
	cfg := newConfig(WithEndpoint("127.0.0.1:abcd"))

	err := validateConfig(cfg)
	assert.Error(t, err)
}

// assert negative sampling rules interval leads to an error.
func TestValidateConfigNegativeDuration(t *testing.T) {
	cfg := newConfig(WithSamplingRulesPollingInterval(-300 * time.Second))

	err := validateConfig(cfg)
	assert.Error(t, err)
}

// assert positive sampling rules interval.
func TestValidateConfigPositiveDuration(t *testing.T) {
	cfg := newConfig(WithSamplingRulesPollingInterval(300 * time.Second))

	err := validateConfig(cfg)
	assert.NoError(t, err)
}