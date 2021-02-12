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

package otelgorm

import (
	oteltrace "go.opentelemetry.io/otel/trace"
)

type config struct {
	dbName         string
	tracerProvider oteltrace.TracerProvider
}

// Option is used to configure the client.
type Option func(*config)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return func(cfg *config) {
		cfg.tracerProvider = provider
	}
}

// WithDBName specified the database name to be used in span names
// since its not possible to extract this information from gorm
func WithDBName(name string) Option {
	return func(cfg *config) {
		cfg.dbName = name
	}
}
