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
	"go.opentelemetry.io/otel/api/trace"
)

// TracedSessionConfig provides configuration for sessions
// created with NewSessionWithTracing.
type TracedSessionConfig struct {
	otelConfig      *OtelConfig
	queryObserver   gocql.QueryObserver
	batchObserver   gocql.BatchObserver
	connectObserver gocql.ConnectObserver
}

// OtelConfig provides OpenTelemetry configuration.
type OtelConfig struct {
	Tracer            trace.Tracer
	InstrumentQuery   bool
	InstrumentBatch   bool
	InstrumentConnect bool
}

// TracedSessionOption is a function type that applies
// a particular configuration to the traced session in question.
type TracedSessionOption func(*TracedSessionConfig)

// ------------------------------------------ TracedSessionOptions

// WithQueryObserver sets an additional QueryObserver to the session configuration. Use this if
// there is an existing QueryObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithQueryObserver(observer gocql.QueryObserver) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.queryObserver = observer
	}
}

// WithBatchObserver sets an additional BatchObserver to the session configuration. Use this if
// there is an existing BatchObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithBatchObserver(observer gocql.BatchObserver) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.batchObserver = observer
	}
}

// WithConnectObserver sets an additional ConnectObserver to the session configuration. Use this if
// there is an existing ConnectObserver that you would like called. It will be called after the
// OpenTelemetry implementation, if it is not nil. Defaults to nil.
func WithConnectObserver(observer gocql.ConnectObserver) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.connectObserver = observer
	}
}

// ------------------------------------------ Otel Options

// WithTracer will set tracer to be the tracer used to create spans
// for query, batch query, and connection instrumentation.
// Defaults to global.Tracer("github.com/gocql/gocql").
func WithTracer(tracer trace.Tracer) TracedSessionOption {
	return func(c *TracedSessionConfig) {
		c.otelConfig.Tracer = tracer
	}
}

// WithQueryInstrumentation will enable and disable instrumentation of
// queries. Defaults to enabled.
func WithQueryInstrumentation(enabled bool) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.otelConfig.InstrumentQuery = enabled
	}
}

// WithBatchInstrumentation will enable and disable insturmentation of
// batch queries. Defaults to enabled.
func WithBatchInstrumentation(enabled bool) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.otelConfig.InstrumentBatch = enabled
	}
}

// WithConnectInstrumentation will enable and disable instrumentation of
// connection attempts. Defaults to enabled.
func WithConnectInstrumentation(enabled bool) TracedSessionOption {
	return func(cfg *TracedSessionConfig) {
		cfg.otelConfig.InstrumentConnect = enabled
	}
}

// ------------------------------------------ Private Functions

func configure(options ...TracedSessionOption) *TracedSessionConfig {
	config := &TracedSessionConfig{
		otelConfig: otelConfiguration(),
	}

	for _, apply := range options {
		apply(config)
	}

	return config
}

func otelConfiguration() *OtelConfig {
	return &OtelConfig{
		Tracer:            global.Tracer("github.com/gocql/gocql"),
		InstrumentQuery:   true,
		InstrumentBatch:   true,
		InstrumentConnect: true,
	}
}
