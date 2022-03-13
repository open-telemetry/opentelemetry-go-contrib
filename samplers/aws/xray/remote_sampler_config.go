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

package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const (
	defaultProxyEndpoint   = "127.0.0.1:2000"
	defaultPollingInterval = 300
)

type config struct {
	endpoint                     string
	samplingRulesPollingInterval time.Duration
	logger                       logr.Logger
}

// Option sets configuration on the sampler.
type Option interface {
	apply(*config) *config
}

type optionFunc func(*config) *config

func (f optionFunc) apply(cfg *config) *config {
	return f(cfg)
}

// WithEndpoint sets custom proxy endpoint.
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.endpoint = endpoint
		return cfg
	})
}

// WithSamplingRulesPollingInterval sets polling interval for sampling rules.
func WithSamplingRulesPollingInterval(polingInterval time.Duration) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.samplingRulesPollingInterval = polingInterval
		return cfg
	})
}

// WithLogger sets custom logging for remote sampling implementation.
func WithLogger(l logr.Logger) Option {
	return optionFunc(func(cfg *config) *config {
		cfg.logger = l
		return cfg
	})
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		endpoint:                     defaultProxyEndpoint,
		samplingRulesPollingInterval: defaultPollingInterval * time.Second,
		logger:                       stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error}),
	}

	for _, option := range opts {
		option.apply(cfg)
	}

	return cfg
}

func validateConfig(cfg *config) (err error) {
	// check endpoint follows certain format
	split := strings.Split(cfg.endpoint, ":")

	if len(split) > 2 {
		return fmt.Errorf("endpoint validation error: expected format is 127.0.0.1:8080")
	}

	// validate host name
	r, err := regexp.Compile("[^A-Za-z0-9.]")
	if err != nil {
		return err
	}

	if r.MatchString(split[0]) {
		return fmt.Errorf("endpoint validation error: expected format is 127.0.0.1:8080")
	}

	// validate port
	if _, err := strconv.Atoi(split[1]); err != nil {
		return fmt.Errorf("endpoint validation error: expected format is 127.0.0.1:8080")
	}

	// validate polling interval is positive
	if math.Signbit(float64(cfg.samplingRulesPollingInterval)) {
		return fmt.Errorf("endpoint validation error: samplingRulesPollingInterval should be positive number")
	}

	return
}
