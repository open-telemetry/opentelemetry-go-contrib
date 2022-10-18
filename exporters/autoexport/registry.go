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

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	otelExporterOTLPProtoEnvKey = "OTEL_EXPORTER_OTLP_PROTOCOL"
)

// registry maintains a map of exporter names to SpanExporter
// implementations that is safe for concurrent use by multiple goroutines
// without additional locking or coordination.
type registry struct {
	mu    sync.Mutex
	names map[string]trace.SpanExporter
}

var (
	// envRegistry is the index of all supported exporter types
	// and their mapping to a SpanExporter.
	envRegistry = &registry{
		names: map[string]trace.SpanExporter{},
	}

	// errUnknownExpoter is returned when an unknown exporter name is used in
	// the OTEL_*_EXPORTER environment variables.
	errUnknownExpoter = errors.New("unknown exporter")

	// errInvalidOtlpProtocol is returned when an invalid protocol is used in
	// the OTEL_EXPORTER_OTLP_PROTOCOL environment variable.
	errInvalidOTLPProtocol = errors.New("invalid OTLP protocol - should be one of ['grpc', 'http/protobuf']")

	// errDuplicateRegistration is returned when an duplicate registration is detected.
	errDuplicateRegistration = errors.New("duplicate registration")
)

// load returns the value stored in the registry index for a key, or nil if no
// value is present. The ok result indicates whether value was found in the
// index.
func (r *registry) load(key string) (p trace.SpanExporter, ok bool) {
	r.mu.Lock()
	p, ok = r.names[key]
	r.mu.Unlock()
	return p, ok
}

// store sets the value for a key if is not already in the registry. errDuplicateRegistration
// is returned if the registry already contains key.
func (r *registry) store(key string, value trace.SpanExporter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.names == nil {
		r.names = map[string]trace.SpanExporter{key: value}
		return nil
	}
	if _, ok := r.names[key]; ok {
		return fmt.Errorf("%w: %q", errDuplicateRegistration, key)
	}
	r.names[key] = value
	return nil
}

// drop removes key from the registry if it exists, otherwise nothing.
func (r *registry) drop(key string) {
	r.mu.Lock()
	delete(r.names, key)
	r.mu.Unlock()
}

// RegisterSpanExporter sets the SpanExporter e to be used when the
// OTEL_TRACES_EXPORTERS environment variable contains the exporter name. This
// will panic if name has already been registered.
func RegisterSpanExporter(name string, e trace.SpanExporter) {
	if err := envRegistry.store(name, e); err != nil {
		// envRegistry.store will return errDuplicateRegistration if name is already
		// registered. Panic here so the user is made aware of the duplicate
		// registration, which could be done by malicious code trying to
		// intercept cross-cutting concerns.
		//
		// Panic for all other errors as well. At this point there should not
		// be any other errors returned from the store operation. If there
		// are, alert the developer that adding them as soon as possible that
		// they need to be handled here.
		panic(err)
	}
}

// SpanExporter returns a SpanExporter using passed in name
// using the list of registered SpanExporters. Each name must match an
// already registered SpanExporter or a default (otlp).
//
// An error is returned for any unknown exporters.
func SpanExporter(name string) (trace.SpanExporter, error) {
	if name == "otlp" {
		proto := "grpc"
		if protoStr, ok := os.LookupEnv(otelExporterOtlpProtoEnvKey); ok {
			proto = protoStr
		}

		switch proto {
		case "grpc":
			return otlptracegrpc.New(context.Background())
		case "http/protobuf":
			return otlptracehttp.New(context.Background())
		default:
			return nil, errInvalidOtlpProtocol
		}
	}

	exp, ok := envRegistry.load(name)
	if !ok {
		return nil, errUnknownExpoter
	}
	return exp, nil
}
