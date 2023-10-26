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
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MetricOption applies an autoexport configuration option.
type MetricOption = option[metric.Reader]

// WithFallbackMetricReader sets the fallback exporter to use when no exporter
// is configured through the OTEL_METRICS_EXPORTER environment variable.
func WithFallbackMetricReader(exporter metric.Reader) MetricOption {
	return withFallback[metric.Reader](exporter)
}

// NewMetricReader returns a configured [go.opentelemetry.io/otel/sdk/metric.Reader]
// defined using the environment variables described below.
//
// OTEL_METRICS_EXPORTER defines the metrics exporter; supported values:
//   - "none" - "no operation" exporter
//   - "otlp" (default) - OTLP exporter; see [go.opentelemetry.io/otel/exporters/otlp/otlpmetric]
//
// OTEL_EXPORTER_OTLP_PROTOCOL defines OTLP exporter's transport protocol;
// supported values:
//   - "grpc" - protobuf-encoded data using gRPC wire format over HTTP/2 connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc]
//   - "http/protobuf" (default) -  protobuf-encoded data over HTTP connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp]
//
// An error is returned if an environment value is set to an unhandled value.
//
// Use [RegisterMetricReader] to handle more values of OTEL_METRICS_EXPORTER.
//
// Use [WithFallbackMetricReader] option to change the returned exporter
// when OTEL_TRACES_EXPORTER is unset or empty.
//
// Use [IsNoneMetricReader] to check if the retured exporter is a "no operation" exporter.
func NewMetricReader(ctx context.Context, opts ...MetricOption) (metric.Reader, error) {
	return metricsSignal.create(ctx, opts...)
}

// RegisterMetricReader sets the MetricReader factory to be used when the
// OTEL_METRICS_EXPORTERS environment variable contains the exporter name. This
// will panic if name has already been registered.
func RegisterMetricReader(name string, factory func(context.Context) (metric.Reader, error)) {
	must(metricsSignal.registry.store(name, factory))
}

var metricsSignal = newSignal[metric.Reader]("OTEL_METRICS_EXPORTER")

func init() {
	RegisterMetricReader("otlp", func(ctx context.Context) (metric.Reader, error) {
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
	})
	RegisterMetricReader("none", func(ctx context.Context) (metric.Reader, error) {
		return newNoopMetricReader(), nil
	})
	RegisterMetricReader("prometheus", func(ctx context.Context) (metric.Reader, error) {
		// create an isolated registry instead of using the global registry --
		// the user might not want to mix OTel with non-OTel metrics
		reg := prometheus.NewRegistry()

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		server := http.Server{
			// Timeouts are necessary to make a server resilent to attacks, but ListenAndServe doesn't set any.
			// We use values from this example: https://blog.cloudflare.com/exposing-go-on-the-internet/#:~:text=There%20are%20three%20main%20timeouts
			// TODO: find defaults with a better justification, and/or make these configurable
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      mux,
		}

		// environment variable names and defaults specified at https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#prometheus-exporter
		host := getenv("OTEL_EXPORTER_PROMETHEUS_HOST", "localhost")
		port := getenv("OTEL_EXPORTER_PROMETHEUS_PORT", "9464")
		addr := host + ":" + port
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			// TODO: retry binding, to handle process restarts
			otel.Handle(fmt.Errorf("unable to bind address %s for Prometheus exporter: %w", addr, err))
		}

		reader, err := promexporter.New(promexporter.WithRegisterer(reg))
		if err != nil {
			return nil, err
		}

		go func() {
			otel.Handle(server.Serve(lis))
		}()

		return readerWithServer{reader, &server}, err
	})
}

type readerWithServer struct {
	metric.Reader
	server *http.Server
}

func (rws readerWithServer) Shutdown(ctx context.Context) error {
	return errors.Join(
		rws.Reader.Shutdown(ctx),
		rws.server.Shutdown(ctx),
	)
}

func getenv(key, fallback string) string {
	result, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return result
}
