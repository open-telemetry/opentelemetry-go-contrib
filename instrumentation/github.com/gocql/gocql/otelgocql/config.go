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

package otelgocql // import "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql"

import (
	"github.com/gocql/gocql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	tracerProvider    trace.TracerProvider
	meterProvider     metric.MeterProvider
	instrumentQuery   bool
	instrumentBatch   bool
	instrumentConnect bool
	queryObserver     gocql.QueryObserver
	batchObserver     gocql.BatchObserver
	connectObserver   gocql.ConnectObserver
}

// Option applies a configuration option.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithQueryObserver sets an additional QueryObserver to the session configuration. Use this if
// there is an existing QueryObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithQueryObserver(observer gocql.QueryObserver) Option {
	return optionFunc(func(cfg *config) {
		cfg.queryObserver = observer
	})
}

// WithBatchObserver sets an additional BatchObserver to the session configuration. Use this if
// there is an existing BatchObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithBatchObserver(observer gocql.BatchObserver) Option {
	return optionFunc(func(cfg *config) {
		cfg.batchObserver = observer
	})
}

// WithConnectObserver sets an additional ConnectObserver to the session configuration. Use this if
// there is an existing ConnectObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithConnectObserver(observer gocql.ConnectObserver) Option {
	return optionFunc(func(cfg *config) {
		cfg.connectObserver = observer
	})
}

// WithTracerProvider will set the trace provider used to get a tracer
// for creating spans. Defaults to TracerProvider().
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.tracerProvider = provider
		}
	})
}

// WithMeterProvider will set the meter provider used to get a meter
// for creating instruments.
// Defaults to global.GetMeterProvider().
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.meterProvider = provider
		}
	})
}

// WithQueryInstrumentation will enable and disable instrumentation of
// queries. Defaults to enabled.
func WithQueryInstrumentation(enabled bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.instrumentQuery = enabled
	})
}

// WithBatchInstrumentation will enable and disable insturmentation of
// batch queries. Defaults to enabled.
func WithBatchInstrumentation(enabled bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.instrumentBatch = enabled
	})
}

// WithConnectInstrumentation will enable and disable instrumentation of
// connection attempts. Defaults to enabled.
func WithConnectInstrumentation(enabled bool) Option {
	return optionFunc(func(cfg *config) {
		cfg.instrumentConnect = enabled
	})
}

func newConfig(options ...Option) *config {
	cfg := &config{
		tracerProvider:    otel.GetTracerProvider(),
		meterProvider:     global.MeterProvider(),
		instrumentQuery:   true,
		instrumentBatch:   true,
		instrumentConnect: true,
	}

	for _, apply := range options {
		apply.apply(cfg)
	}

	return cfg
}
