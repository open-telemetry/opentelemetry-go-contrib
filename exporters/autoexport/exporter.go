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

	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	otelTracesExportersEnvKey = "OTEL_TRACES_EXPORTER"
)

type config struct {
	fallbackExporter trace.SpanExporter
}

func newConfig(ctx context.Context, opts ...Option) (config, error) {
	cfg := config{}
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}

	// if no fallback exporter is configured, use otlp exporter
	if cfg.fallbackExporter == nil {
		exp, err := spanExporter(context.Background(), "otlp")
		if err != nil {
			return cfg, err
		}
		cfg.fallbackExporter = exp
	}
	return cfg, nil
}

// Option applies an autoexport configuration option.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (fn optionFunc) apply(cfg config) config {
	return fn(cfg)
}

// WithFallbackSpanExporter sets the fallback exporter to use when no exporter
// is configured through the OTEL_TRACES_EXPORTER environment variable.
func WithFallbackSpanExporter(exporter trace.SpanExporter) Option {
	return optionFunc(func(cfg config) config {
		cfg.fallbackExporter = exporter
		return cfg
	})
}

// NewSpanExporter returns a configured [go.opentelemetry.io/otel/sdk/trace.SpanExporter]
// defined using the environment variables described below.
//
// OTEL_TRACES_EXPORTER defines the traces exporter; supported values:
//   - "none" - "no operation" exporter
//   - "otlp" (default) - OTLP exporter
//
// OTEL_EXPORTER_OTLP_PROTOCOL defines OTLP exporter's transport protocol;
// supported values:
//   - "grpc" - protobuf-encoded data using gRPC wire format over HTTP/2 connection
//   - "http/protobuf" (default) -  protobuf-encoded data over HTTP connection
//
// OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_TRACES_ENDPOINT define
// the target URL to which the OTLP exporter is going to send telemetry.
// For OTEL_EXPORTER_OTLP_ENDPOINT and http/protobuf protocol,
// "v1/traces" is appended to the provided value.
// Default value for grpc protocol: "http://localhost:4317" .
// Default value for http/protobuf protocol: "http://localhost:4318[/v1/traces]".
//
// OTEL_EXPORTER_OTLP_CERTIFICATE, OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE define
// a filepath to a TLS certificate pool to use by OTLP exporter when verifying
// a server's TLS credentials. If it exists, it is parsed as a [crypto/x509.CertPool].
// Default value: "".
//
// OTEL_EXPORTER_OTLP_HEADERS, OTEL_EXPORTER_OTLP_TRACES_HEADERS define
// a comma-separated list of additional HTTP headers sent by OTLP exporter,
// for example: Authorization=secret,X-Key=Value.
// Default value: "".
//
// OTEL_EXPORTER_OTLP_COMPRESSION, OTEL_EXPORTER_OTLP_TRACES_COMPRESSION define
// the compression used by OTLP exporter; supported values:
//   - "gzip"
//   - "" (default).
//
// OTEL_EXPORTER_OTLP_TIMEOUT, OTEL_EXPORTER_OTLP_TRACES_TIMEOUT define
// the maximum time the OTLP exporter will wait for each batch export.
// The value is interpreted as number of milliseconds.
// Default value: "10000" (10 seconds).
//
// OTEL_EXPORTER_OTLP_TRACES_* environment variables have precedence
// over OTEL_EXPORTER_OTLP_* environment variables.
//
// An error is returned if an environment value is set to an unhandled value.
//
// Use [RegisterSpanExporter] to handle more values of OTEL_TRACES_EXPORTER.
//
// Use [WithFallbackSpanExporter] option to change the returned exporter
// when OTEL_TRACES_EXPORTER is unset or empty.
//
// Use [IsNone] to check if the retured exporter is a "no operation" exporter.
func NewSpanExporter(ctx context.Context, opts ...Option) (trace.SpanExporter, error) {
	// prefer exporter configured via environment variables over exporter
	// passed in via exporter parameter
	envExporter, err := makeExporterFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if envExporter != nil {
		return envExporter, nil
	}
	config, err := newConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return config.fallbackExporter, nil
}

// makeExporterFromEnv returns a configured SpanExporter defined by the OTEL_TRACES_EXPORTER
// environment variable.
// nil is returned if no exporter is defined for the environment variable.
func makeExporterFromEnv(ctx context.Context) (trace.SpanExporter, error) {
	expType := os.Getenv(otelTracesExportersEnvKey)
	if expType == "" {
		return nil, nil
	}
	return spanExporter(ctx, expType)
}
