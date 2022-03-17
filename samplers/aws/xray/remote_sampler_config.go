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
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const (
	defaultPollingInterval = 300
)

type config struct {
	endpoint                     url.URL
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
func WithEndpoint(endpoint url.URL) Option {
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

func newConfig(opts ...Option) (*config, error) {
	defaultProxyEndpoint, err := url.Parse("http://127.0.0.1:2000")
	if err != nil {
		return nil, err
	}

	cfg := &config{
		endpoint:                     *defaultProxyEndpoint,
		samplingRulesPollingInterval: defaultPollingInterval * time.Second,
		logger:                       stdr.NewWithOptions(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), stdr.Options{LogCaller: stdr.Error}),
	}

	for _, option := range opts {
		option.apply(cfg)
	}

	// validate config
	err = validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateConfig(cfg *config) (err error) {
	// check endpoint follows certain format
	endpointHostSplit := strings.Split(cfg.endpoint.Host, ":")

	if len(endpointHostSplit) > 2 {
		return fmt.Errorf("config validation error: expected endpoint host format is hostname:port")
	}

	hostName := endpointHostSplit[0]

	// validate host name
	r, err := regexp.Compile("[^A-Za-z0-9.]")
	if err != nil {
		return err
	}

	if r.MatchString(hostName) || hostName == "" {
		return fmt.Errorf("config validation error: host name should not contain special characters or empty")
	}

	// validate polling interval is positive
	if math.Signbit(float64(cfg.samplingRulesPollingInterval)) {
		return fmt.Errorf("config validation error: samplingRulesPollingInterval should be positive number")
	}

	return
}
