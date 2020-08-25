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

package gocql

import (
	"github.com/gocql/gocql"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

// TracedSessionConfig provides configuration for sessions
// created with NewSessionWithTracing.
type TracedSessionConfig struct {
	tracerProvider    trace.Provider
	meterProvider     metric.Provider
	instrumentQuery   bool
	instrumentBatch   bool
	instrumentConnect bool
	queryObserver     gocql.QueryObserver
	batchObserver     gocql.BatchObserver
	connectObserver   gocql.ConnectObserver
}

// TracedSessionOption applies a configuration option to
// the given TracedSessionConfig.
type TracedSessionOption interface {
	Apply(*TracedSessionConfig)
}

// TracedSessionOptionFunc is a function type that applies
// a particular configuration to the traced session in question.
type TracedSessionOptionFunc func(*TracedSessionConfig)

// Apply will apply the TracedSessionOptionFunc to c, the given
// TracedSessionConfig.
func (o TracedSessionOptionFunc) Apply(c *TracedSessionConfig) {
	o(c)
}

// ------------------------------------------ TracedSessionOptions

// WithQueryObserver sets an additional QueryObserver to the session configuration. Use this if
// there is an existing QueryObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithQueryObserver(observer gocql.QueryObserver) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.queryObserver = observer
	})
}

// WithBatchObserver sets an additional BatchObserver to the session configuration. Use this if
// there is an existing BatchObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithBatchObserver(observer gocql.BatchObserver) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.batchObserver = observer
	})
}

// WithConnectObserver sets an additional ConnectObserver to the session configuration. Use this if
// there is an existing ConnectObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithConnectObserver(observer gocql.ConnectObserver) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.connectObserver = observer
	})
}

// WithTracerProvider will set the trace provider used to get a tracer
// for creating spans. Defaults to global.TraceProvider()
func WithTracerProvider(provider trace.Provider) TracedSessionOption {
	return TracedSessionOptionFunc(func(c *TracedSessionConfig) {
		c.tracerProvider = provider
	})
}

// WithMeterProvider will set the meter provider used to get a meter
// for creating instruments.
// Defaults to global.MeterProvider().
func WithMeterProvider(provider metric.Provider) TracedSessionOption {
	return TracedSessionOptionFunc(func(c *TracedSessionConfig) {
		c.meterProvider = provider
	})
}

// WithQueryInstrumentation will enable and disable instrumentation of
// queries. Defaults to enabled.
func WithQueryInstrumentation(enabled bool) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.instrumentQuery = enabled
	})
}

// WithBatchInstrumentation will enable and disable insturmentation of
// batch queries. Defaults to enabled.
func WithBatchInstrumentation(enabled bool) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.instrumentBatch = enabled
	})
}

// WithConnectInstrumentation will enable and disable instrumentation of
// connection attempts. Defaults to enabled.
func WithConnectInstrumentation(enabled bool) TracedSessionOption {
	return TracedSessionOptionFunc(func(cfg *TracedSessionConfig) {
		cfg.instrumentConnect = enabled
	})
}

// ------------------------------------------ Private Functions

func configure(options ...TracedSessionOption) *TracedSessionConfig {
	config := &TracedSessionConfig{
		tracerProvider:    global.TraceProvider(),
		meterProvider:     global.MeterProvider(),
		instrumentQuery:   true,
		instrumentBatch:   true,
		instrumentConnect: true,
	}

	for _, apply := range options {
		apply.Apply(config)
	}

	return config
}
