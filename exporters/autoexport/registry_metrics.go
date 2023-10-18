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

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
)

func newMetricReaderRegistry() registry[metric.Reader] {
	return registry[metric.Reader]{
		names: map[string]func(context.Context) (metric.Reader, error){
			"":     buildOTLPMetricReader,
			"otlp": buildOTLPMetricReader,
			"none": func(ctx context.Context) (metric.Reader, error) { return noopMetricReader, nil },
		},
	}
}

// metricReaderRegistry is the package level registry of exporter registrations
// and their mapping to a MetricReader factory func(context.Context) (metric.MetricReader, error).
var metricReaderRegistry = newMetricReaderRegistry()

// RegisterMetricReader sets the MetricReader factory to be used when the
// OTEL_METRICS_EXPORTERS environment variable contains the exporter name. This
// will panic if name has already been registered.
func RegisterMetricReader(name string, factory func(context.Context) (metric.Reader, error)) {
	if err := metricReaderRegistry.store(name, factory); err != nil {
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

// metricReader returns a metric reader using the passed in name
// from the list of registered metric.Readers. Each name must match an
// already registered metric.Reader. A default OTLP exporter is registered
// under both an empty string "" and "otlp".
// An error is returned for any unknown exporters.
func metricReader(ctx context.Context, name string) (metric.Reader, error) {
	exp, err := metricReaderRegistry.load(ctx, name)
	if err != nil {
		return nil, err
	}
	return exp, nil
}

// buildOTLPMetricReader creates an OTLP metric reader using the environment variable
// OTEL_EXPORTER_OTLP_PROTOCOL to determine the exporter protocol.
// Defaults to http/protobuf protocol.
func buildOTLPMetricReader(ctx context.Context) (metric.Reader, error) {
	proto := os.Getenv(otelExporterOTLPProtoEnvKey)
	if proto == "" {
		proto = "http/protobuf"
	}

	switch proto {
	case "grpc":
		r, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		return metric.NewPeriodicReader(r), nil
	case "http/protobuf":
		r, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return nil, err
		}
		return metric.NewPeriodicReader(r), nil
	default:
		return nil, errInvalidOTLPProtocol
	}
}
