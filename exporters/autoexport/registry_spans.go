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
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

func newSpanExporterRegistry() registry[trace.SpanExporter] {
	return registry[trace.SpanExporter]{
		names: map[string]func(context.Context) (trace.SpanExporter, error){
			"":     buildOTLPSpanExporter,
			"otlp": buildOTLPSpanExporter,
			"none": func(ctx context.Context) (trace.SpanExporter, error) { return noop{}, nil },
		},
	}
}

// spanExporterRegistry is the package level registry of exporter registrations
// and their mapping to a SpanExporter factory func(context.Context) (trace.SpanExporter, error).
var spanExporterRegistry = newSpanExporterRegistry()

// RegisterSpanExporter sets the SpanExporter factory to be used when the
// OTEL_TRACES_EXPORTERS environment variable contains the exporter name. This
// will panic if name has already been registered.
func RegisterSpanExporter(name string, factory func(context.Context) (trace.SpanExporter, error)) {
	if err := spanExporterRegistry.store(name, factory); err != nil {
		// registry.store will return errDuplicateRegistration if name is already
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

// spanExporter returns a span exporter using the passed in name
// from the list of registered SpanExporters. Each name must match an
// already registered SpanExporter. A default OTLP exporter is registered
// under both an empty string "" and "otlp".
// An error is returned for any unknown exporters.
func spanExporter(ctx context.Context, name string) (trace.SpanExporter, error) {
	exp, err := spanExporterRegistry.load(ctx, name)
	if err != nil {
		return nil, err
	}
	return exp, nil
}

// buildOTLPSpanExporter creates an OTLP span exporter using the environment variable
// OTEL_EXPORTER_OTLP_PROTOCOL to determine the exporter protocol.
// Defaults to http/protobuf protocol.
func buildOTLPSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	proto := os.Getenv(otelExporterOTLPProtoEnvKey)
	if proto == "" {
		proto = "http/protobuf"
	}

	switch proto {
	case "grpc":
		return otlptracegrpc.New(ctx)
	case "http/protobuf":
		return otlptracehttp.New(ctx)
	default:
		return nil, errInvalidOTLPProtocol
	}
}
