// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	v1 "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/otelconf/internal/testtls"
)

func TestMeterProvider(t *testing.T) {
	tests := []struct {
		name         string
		cfg          configOptions
		wantProvider metric.MeterProvider
		wantErr      error
	}{
		{
			name:         "no-meter-provider-configured",
			wantProvider: noop.NewMeterProvider(),
		},
		{
			name: "error-in-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					MeterProvider: &MeterProvider{
						Readers: []MetricReader{
							{
								Periodic: &PeriodicMetricReader{},
								Pull:     &PullMetricReader{},
							},
						},
					},
				},
			},
			wantProvider: noop.NewMeterProvider(),
			wantErr:      newErrInvalid("must not specify multiple metric reader type"),
		},
		{
			name: "multiple-errors-in-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					MeterProvider: &MeterProvider{
						Readers: []MetricReader{
							{
								Periodic: &PeriodicMetricReader{},
								Pull:     &PullMetricReader{},
							},
							{
								Periodic: &PeriodicMetricReader{
									Exporter: PushMetricExporter{
										Console:  &ConsoleMetricExporter{},
										OTLPGrpc: &OTLPGrpcMetricExporter{},
									},
								},
							},
						},
					},
				},
			},
			wantProvider: noop.NewMeterProvider(),
			wantErr:      newErrInvalid("must not specify multiple metric reader type"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp, shutdown, err := meterProvider(tt.cfg, resource.Default())
			require.Equal(t, tt.wantProvider, mp)
			assert.ErrorIs(t, err, tt.wantErr)
			require.NoError(t, shutdown(t.Context()))
		})
	}
}

func TestMeterProviderOptions(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	defer srv.Close()

	cfg := OpenTelemetryConfiguration{
		MeterProvider: &MeterProvider{
			Readers: []MetricReader{{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint: new(srv.URL),
						},
					},
				},
			}},
		},
	}

	var buf bytes.Buffer
	stdoutmetricExporter, err := stdoutmetric.New(stdoutmetric.WithWriter(&buf))
	require.NoError(t, err)

	res := resource.NewSchemaless(attribute.String("foo", "bar"))
	sdk, err := NewSDK(
		WithOpenTelemetryConfiguration(cfg),
		WithMeterProviderOptions(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(stdoutmetricExporter))),
		WithMeterProviderOptions(sdkmetric.WithResource(res)),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, sdk.Shutdown(t.Context()))
		// The exporter, which we passed in as an extra option to NewSDK,
		// should be wired up to the provider in addition to the
		// configuration-based OTLP exporter.
		assert.NotZero(t, buf)
		assert.Equal(t, 1, calls) // flushed on shutdown

		// Options provided by WithMeterProviderOptions may be overridden
		// by configuration, e.g. the resource is always defined via
		// configuration.
		assert.NotContains(t, buf.String(), "foo")
	}()

	counter, _ := sdk.MeterProvider().Meter("test").Int64Counter("counter")
	counter.Add(t.Context(), 1)
}

func TestReader(t *testing.T) {
	material := testtls.Write(t)
	consoleExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := t.Context()
	otlpGRPCExporter, err := otlpmetricgrpc.New(ctx)
	require.NoError(t, err)
	otlpHTTPExporter, err := otlpmetrichttp.New(ctx)
	require.NoError(t, err)
	testCases := []struct {
		name       string
		reader     MetricReader
		args       any
		wantErrT   error
		wantReader sdkmetric.Reader
	}{
		{
			name:     "no reader",
			wantErrT: newErrInvalid("no valid metric reader"),
		},
		{
			name: "pull/no-exporter",
			reader: MetricReader{
				Pull: &PullMetricReader{},
			},
			wantErrT: newErrInvalid("no valid metric exporter"),
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("http://localhost:4318"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-exporter-with-path",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("http://localhost:4318/path/123"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-good-ca-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("https://localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &GrpcTls{
								CaFile: new(material.CACertPath),
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-bad-ca-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("https://localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &GrpcTls{
								CaFile: new(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "periodic/otlp-grpc-bad-client-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &GrpcTls{
								KeyFile:  new(material.BadCertPath),
								CertFile: new(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "periodic/otlp-grpc-bad-headerslist",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							HeadersList: new("==="),
						},
					},
				},
			},
			wantErrT: newErrInvalid("invalid headers_list"),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-exporter-socket-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("unix:collector.sock"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-scheme",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-invalid-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new(" "),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("endpoint parsing failed"),
		},
		{
			name: "periodic/otlp-grpc-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-delta-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceDelta),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-cumulative-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceCumulative),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-lowmemory-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceLowMemory),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-invalid-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: (*ExporterTemporalityPreference)(new("invalid")),
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported temporality preference \"invalid\""),
		},
		{
			name: "periodic/otlp-grpc-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("invalid"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/otlp-http-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("http://localhost:4318"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-good-ca-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("https://localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &HttpTls{
								CaFile: new(material.CACertPath),
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-bad-ca-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("https://localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &HttpTls{
								CaFile: new(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "periodic/otlp-http-bad-client-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Tls: &HttpTls{
								KeyFile:  new(material.BadCertPath),
								CertFile: new(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "periodic/otlp-http-bad-headerslist",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4317"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							HeadersList: new("==="),
						},
					},
				},
			},
			wantErrT: newErrInvalid("invalid headers_list"),
		},
		{
			name: "periodic/otlp-http-exporter-with-path",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("http://localhost:4318/path/123"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-exporter-no-scheme",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-invalid-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new(" "),
							Compression: new("gzip"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("endpoint parsing failed"),
		},
		{
			name: "periodic/otlp-http-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-cumulative-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceCumulative),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-lowmemory-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceLowMemory),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-delta-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: new(ExporterTemporalityPreferenceDelta),
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-invalid-temporality",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("none"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
							TemporalityPreference: (*ExporterTemporalityPreference)(new("invalid")),
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported temporality preference \"invalid\""),
		},
		{
			name: "periodic/otlp-http-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint:    new("localhost:4318"),
							Compression: new("invalid"),
							Timeout:     new(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: new("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/otlp-http-invalid-encoding",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							Endpoint: new("http://localhost:4318"),
							Encoding: new(OTLPHttpEncoding("json")),
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported encoding \"json\""),
		},
		{
			name: "periodic/no-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{},
				},
			},
			wantErrT: newErrInvalid("no valid metric exporter"),
		},
		{
			name: "periodic/console-exporter-with-cardinality-limits",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					CardinalityLimits: &CardinalityLimits{
						Counter:                 new(100),
						UpDownCounter:           new(200),
						Histogram:               new(300),
						ObservableCounter:       new(400),
						ObservableUpDownCounter: new(500),
						ObservableGauge:         new(600),
						Gauge:                   new(700),
					},
					Exporter: PushMetricExporter{
						Console: &ConsoleMetricExporter{},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(
				consoleExporter,
				sdkmetric.WithCardinalityLimitSelector(func(ik sdkmetric.InstrumentKind) (int, bool) {
					switch ik {
					case sdkmetric.InstrumentKindCounter:
						return 100, false
					case sdkmetric.InstrumentKindUpDownCounter:
						return 200, false
					case sdkmetric.InstrumentKindHistogram:
						return 300, false
					case sdkmetric.InstrumentKindObservableCounter:
						return 400, false
					case sdkmetric.InstrumentKindObservableUpDownCounter:
						return 500, false
					case sdkmetric.InstrumentKindObservableGauge:
						return 600, false
					case sdkmetric.InstrumentKindGauge:
						return 700, false
					}
					return 0, true
				}),
			),
		},
		{
			name: "periodic/console-exporter-with-default-cardinality-limit",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					CardinalityLimits: &CardinalityLimits{
						Default: new(50),
					},
					Exporter: PushMetricExporter{
						Console: &ConsoleMetricExporter{},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(
				consoleExporter,
				sdkmetric.WithCardinalityLimitSelector(func(sdkmetric.InstrumentKind) (int, bool) {
					return 50, false
				}),
			),
		},
		{
			name: "periodic/console-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						Console: &ConsoleMetricExporter{},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(consoleExporter),
		},
		{
			name: "periodic/console-exporter-with-extra-options",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Interval: new(30_000),
					Timeout:  new(5_000),
					Exporter: PushMetricExporter{
						Console: &ConsoleMetricExporter{},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(
				consoleExporter,
				sdkmetric.WithInterval(30_000*time.Millisecond),
				sdkmetric.WithTimeout(5_000*time.Millisecond),
			),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metricReader(t.Context(), tt.reader)
			require.ErrorIs(t, err, tt.wantErrT)
			if tt.wantReader == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantReader), reflect.TypeOf(got))
				var fieldName string
				switch reflect.TypeOf(tt.wantReader).String() {
				case "*metric.PeriodicReader":
					fieldName = "exporter"
				case "otelconf.readerWithServer":
					fieldName = "Reader"
				default:
					fieldName = "e"
				}
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantReader)).FieldByName(fieldName).Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName(fieldName).Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
				require.NoError(t, got.Shutdown(t.Context()))
			}
		})
	}
}

func TestCardinalityLimitSelector(t *testing.T) {
	allKinds := []sdkmetric.InstrumentKind{
		sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindObservableCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter,
		sdkmetric.InstrumentKindObservableGauge,
		sdkmetric.InstrumentKindGauge,
	}

	t.Run("per-kind limits", func(t *testing.T) {
		cl := &CardinalityLimits{
			Counter:                 new(100),
			UpDownCounter:           new(200),
			Histogram:               new(300),
			ObservableCounter:       new(400),
			ObservableUpDownCounter: new(500),
			ObservableGauge:         new(600),
			Gauge:                   new(700),
		}
		sel := cardinalityLimitSelector(cl)
		expected := map[sdkmetric.InstrumentKind]int{
			sdkmetric.InstrumentKindCounter:                 100,
			sdkmetric.InstrumentKindUpDownCounter:           200,
			sdkmetric.InstrumentKindHistogram:               300,
			sdkmetric.InstrumentKindObservableCounter:       400,
			sdkmetric.InstrumentKindObservableUpDownCounter: 500,
			sdkmetric.InstrumentKindObservableGauge:         600,
			sdkmetric.InstrumentKindGauge:                   700,
		}
		for _, ik := range allKinds {
			limit, fallback := sel(ik)
			assert.Equal(t, expected[ik], limit)
			assert.False(t, fallback)
		}
	})

	t.Run("default limit used when kind not set", func(t *testing.T) {
		cl := &CardinalityLimits{
			Default: new(50),
		}
		sel := cardinalityLimitSelector(cl)
		for _, ik := range allKinds {
			limit, fallback := sel(ik)
			assert.Equal(t, 50, limit)
			assert.False(t, fallback)
		}
	})

	t.Run("per-kind overrides default", func(t *testing.T) {
		cl := &CardinalityLimits{
			Default: new(50),
			Counter: new(100),
		}
		sel := cardinalityLimitSelector(cl)
		limit, fallback := sel(sdkmetric.InstrumentKindCounter)
		assert.Equal(t, 100, limit)
		assert.False(t, fallback)

		limit, fallback = sel(sdkmetric.InstrumentKindGauge)
		assert.Equal(t, 50, limit)
		assert.False(t, fallback)
	})

	t.Run("fallback to provider when no limit set", func(t *testing.T) {
		cl := &CardinalityLimits{}
		sel := cardinalityLimitSelector(cl)
		for _, ik := range allKinds {
			limit, fallback := sel(ik)
			assert.Equal(t, 0, limit)
			assert.True(t, fallback)
		}
	})
}

// TestMetricReaderCardinalityLimitsWired verifies that CardinalityLimits set on
// a PeriodicMetricReader are actually wired into the returned SDK reader.
// It records 3 distinct attribute sets with a per-kind limit of 1; the SDK
// must produce exactly 1 data point (only the overflow bucket) rather than
// 3, which would happen if the selector were never registered. With limit=1
// the overflow slot consumes the entire limit, so no normal data points fit.
func TestMetricReaderCardinalityLimitsWired(t *testing.T) {
	ctx := t.Context()

	reader, err := metricReader(ctx, MetricReader{
		Periodic: &PeriodicMetricReader{
			CardinalityLimits: &CardinalityLimits{
				Counter: new(1),
			},
			Exporter: PushMetricExporter{
				Console: &ConsoleMetricExporter{},
			},
		},
	})
	require.NoError(t, err)

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() { assert.NoError(t, mp.Shutdown(t.Context())) }()

	counter, err := mp.Meter("test").Int64Counter("cardinality.wiring.test")
	require.NoError(t, err)

	// Record 3 distinct attribute sets; with limit=1 the SDK must emit only
	// 1 data point: the overflow bucket (the limit counts the overflow slot
	// itself, so no "normal" data points fit alongside it).
	counter.Add(ctx, 1, metric.WithAttributes(attribute.Int("k", 1)))
	counter.Add(ctx, 1, metric.WithAttributes(attribute.Int("k", 2)))
	counter.Add(ctx, 1, metric.WithAttributes(attribute.Int("k", 3)))

	pr := reader.(*sdkmetric.PeriodicReader)
	var rm metricdata.ResourceMetrics
	require.NoError(t, pr.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
	dataPoints := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Sum[int64]).DataPoints
	assert.Len(t, dataPoints, 1)
}

func TestView(t *testing.T) {
	testCases := []struct {
		name            string
		view            View
		args            any
		wantErr         string
		matchInstrument *sdkmetric.Instrument
		wantStream      sdkmetric.Stream
		wantResult      bool
	}{
		{
			name: "selector/invalid_type",
			view: View{
				Selector: ViewSelector{
					InstrumentType: (*InstrumentType)(new("invalid_type")),
				},
			},
			wantErr: "view_selector: instrument_type: invalid value",
		},
		{
			name: "selector/invalid_type",
			view: View{
				Selector: ViewSelector{},
			},
			wantErr: "view_selector: empty selector not supporter",
		},
		{
			name: "all selectors match",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "test_meter_version",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{Name: "test_name", Unit: "test_unit"},
			wantResult: true,
		},
		{
			name: "all selectors no match name",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "not_match",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "test_meter_version",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "all selectors no match unit",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "not_match",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "test_meter_version",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "all selectors no match kind",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: (*InstrumentType)(new("histogram")),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "test_meter_version",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "all selectors no match meter name",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "not_match",
					Version:   "test_meter_version",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "all selectors no match meter version",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "not_match",
					SchemaURL: "test_schema_url",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "all selectors no match meter schema url",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					InstrumentType: new(InstrumentTypeCounter),
					Unit:           new("test_unit"),
					MeterName:      new("test_meter_name"),
					MeterVersion:   new("test_meter_version"),
					MeterSchemaUrl: new("test_schema_url"),
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name: "test_name",
				Unit: "test_unit",
				Kind: sdkmetric.InstrumentKindCounter,
				Scope: instrumentation.Scope{
					Name:      "test_meter_name",
					Version:   "test_meter_version",
					SchemaURL: "not_match",
				},
			},
			wantStream: sdkmetric.Stream{},
			wantResult: false,
		},
		{
			name: "with stream",
			view: View{
				Selector: ViewSelector{
					InstrumentName: new("test_name"),
					Unit:           new("test_unit"),
				},
				Stream: ViewStream{
					Name:          new("new_name"),
					Description:   new("new_description"),
					AttributeKeys: new(IncludeExclude{Included: []string{"foo", "bar"}}),
					Aggregation:   &Aggregation{Sum: make(SumAggregation)},
				},
			},
			matchInstrument: &sdkmetric.Instrument{
				Name:        "test_name",
				Description: "test_description",
				Unit:        "test_unit",
			},
			wantStream: sdkmetric.Stream{
				Name:        "new_name",
				Description: "new_description",
				Unit:        "test_unit",
				Aggregation: sdkmetric.AggregationSum{},
			},
			wantResult: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := view(tt.view)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				gotStream, gotResult := got(*tt.matchInstrument)
				// Remove filter, since it cannot be compared
				gotStream.AttributeFilter = nil
				require.Equal(t, tt.wantStream, gotStream)
				require.Equal(t, tt.wantResult, gotResult)
			}
		})
	}
}

func TestInstrumentType(t *testing.T) {
	testCases := []struct {
		name     string
		instType *InstrumentType
		wantErr  error
		wantKind sdkmetric.InstrumentKind
	}{
		{
			name:     "nil",
			wantKind: sdkmetric.InstrumentKind(0),
		},
		{
			name:     "counter",
			instType: new(InstrumentTypeCounter),
			wantKind: sdkmetric.InstrumentKindCounter,
		},
		{
			name:     "up_down_counter",
			instType: new(InstrumentTypeUpDownCounter),
			wantKind: sdkmetric.InstrumentKindUpDownCounter,
		},
		{
			name:     "histogram",
			instType: new(InstrumentTypeHistogram),
			wantKind: sdkmetric.InstrumentKindHistogram,
		},
		{
			name:     "observable_counter",
			instType: new(InstrumentTypeObservableCounter),
			wantKind: sdkmetric.InstrumentKindObservableCounter,
		},
		{
			name:     "observable_up_down_counter",
			instType: new(InstrumentTypeObservableUpDownCounter),
			wantKind: sdkmetric.InstrumentKindObservableUpDownCounter,
		},
		{
			name:     "observable_gauge",
			instType: new(InstrumentTypeObservableGauge),
			wantKind: sdkmetric.InstrumentKindObservableGauge,
		},
		{
			name:     "invalid",
			instType: (*InstrumentType)(new("invalid")),
			wantErr:  errors.New("instrument_type: invalid value"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := instrumentKind(tt.instType)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
				require.Zero(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantKind, got)
			}
		})
	}
}

func TestAggregation(t *testing.T) {
	testCases := []struct {
		name            string
		aggregation     *Aggregation
		wantAggregation sdkmetric.Aggregation
	}{
		{
			name:            "nil",
			wantAggregation: nil,
		},
		{
			name:            "empty",
			aggregation:     &Aggregation{},
			wantAggregation: nil,
		},
		{
			name: "Base2ExponentialBucketHistogram empty",
			aggregation: &Aggregation{
				Base2ExponentialBucketHistogram: &Base2ExponentialBucketHistogramAggregation{},
			},
			wantAggregation: sdkmetric.AggregationBase2ExponentialHistogram{
				MaxSize:  0,
				MaxScale: 0,
				NoMinMax: true,
			},
		},
		{
			name: "Base2ExponentialBucketHistogram",
			aggregation: &Aggregation{
				Base2ExponentialBucketHistogram: &Base2ExponentialBucketHistogramAggregation{
					MaxSize:      new(2),
					MaxScale:     new(3),
					RecordMinMax: new(true),
				},
			},
			wantAggregation: sdkmetric.AggregationBase2ExponentialHistogram{
				MaxSize:  2,
				MaxScale: 3,
				NoMinMax: false,
			},
		},
		{
			name: "Default",
			aggregation: &Aggregation{
				Default: make(DefaultAggregation),
			},
			wantAggregation: nil,
		},
		{
			name: "Drop",
			aggregation: &Aggregation{
				Drop: make(DropAggregation),
			},
			wantAggregation: sdkmetric.AggregationDrop{},
		},
		{
			name: "ExplicitBucketHistogram empty",
			aggregation: &Aggregation{
				ExplicitBucketHistogram: &ExplicitBucketHistogramAggregation{},
			},
			wantAggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: nil,
				NoMinMax:   true,
			},
		},
		{
			name: "ExplicitBucketHistogram",
			aggregation: &Aggregation{
				ExplicitBucketHistogram: &ExplicitBucketHistogramAggregation{
					Boundaries:   []float64{1, 2, 3},
					RecordMinMax: new(true),
				},
			},
			wantAggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{1, 2, 3},
				NoMinMax:   false,
			},
		},
		{
			name: "LastValue",
			aggregation: &Aggregation{
				LastValue: make(LastValueAggregation),
			},
			wantAggregation: sdkmetric.AggregationLastValue{},
		},
		{
			name: "Sum",
			aggregation: &Aggregation{
				Sum: make(SumAggregation),
			},
			wantAggregation: sdkmetric.AggregationSum{},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregation(tt.aggregation)
			require.Equal(t, tt.wantAggregation, got)
		})
	}
}

func TestNewIncludeExcludeFilter(t *testing.T) {
	testCases := []struct {
		name          string
		attributeKeys *IncludeExclude
		wantPass      []string
		wantFail      []string
	}{
		{
			name:          "empty",
			attributeKeys: nil,
			wantPass:      []string{"foo", "bar"},
			wantFail:      nil,
		},
		{
			name: "filter-with-include",
			attributeKeys: new(IncludeExclude{
				Included: []string{"foo"},
			}),
			wantPass: []string{"foo"},
			wantFail: []string{"bar"},
		},
		{
			name: "filter-with-exclude",
			attributeKeys: new(IncludeExclude{
				Excluded: []string{"foo"},
			}),
			wantPass: []string{"bar"},
			wantFail: []string{"foo"},
		},
		{
			name: "filter-with-include-and-exclude",
			attributeKeys: new(IncludeExclude{
				Included: []string{"bar"},
				Excluded: []string{"foo"},
			}),
			wantPass: []string{"bar"},
			wantFail: []string{"foo"},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newIncludeExcludeFilter(tt.attributeKeys)
			require.NoError(t, err)
			for _, pass := range tt.wantPass {
				require.True(t, got(attribute.KeyValue{Key: attribute.Key(pass), Value: attribute.StringValue("")}))
			}
			for _, fail := range tt.wantFail {
				require.False(t, got(attribute.KeyValue{Key: attribute.Key(fail), Value: attribute.StringValue("")}))
			}
		})
	}
}

func TestNewIncludeExcludeFilterError(t *testing.T) {
	_, err := newIncludeExcludeFilter(new(IncludeExclude{
		Included: []string{"foo"},
		Excluded: []string{"foo"},
	}))
	require.Equal(t, fmt.Errorf("attribute cannot be in both include and exclude list: foo"), err)
}

func Test_otlpGRPCMetricExporter(t *testing.T) {
	material := testtls.Write(t)
	type args struct {
		ctx        context.Context
		otlpConfig *OTLPGrpcMetricExporter
	}
	tests := []struct {
		name           string
		args           args
		grpcServerOpts func() ([]grpc.ServerOption, error)
	}{
		{
			name: "no TLS config",
			args: args{
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcMetricExporter{
					Compression: new("gzip"),
					Timeout:     new(50000),
					Tls: &GrpcTls{
						Insecure: new(true),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: new("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				return []grpc.ServerOption{}, nil
			},
		},
		{
			name: "with TLS config",
			args: args{
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcMetricExporter{
					Compression: new("gzip"),
					Timeout:     new(50000),
					Tls: &GrpcTls{
						CaFile: new(material.CACertPath),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: new("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				tlsCreds, err := credentials.NewServerTLSFromFile(material.ServerCertPath, material.ServerKeyPath)
				if err != nil {
					return nil, err
				}
				opts = append(opts, grpc.Creds(tlsCreds))
				return opts, nil
			},
		},
		{
			name: "with TLS config and client key",
			args: args{
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcMetricExporter{
					Compression: new("gzip"),
					Timeout:     new(50000),
					Tls: &GrpcTls{
						CaFile:   new(material.CACertPath),
						KeyFile:  new(material.ClientKeyPath),
						CertFile: new(material.ClientCertPath),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: new("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				cert, err := tls.LoadX509KeyPair(material.ServerCertPath, material.ServerKeyPath)
				if err != nil {
					return nil, err
				}
				caCert, err := os.ReadFile(material.CACertPath)
				if err != nil {
					return nil, err
				}
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsCreds := credentials.NewTLS(&tls.Config{
					Certificates: []tls.Certificate{cert},
					ClientCAs:    caCertPool,
					ClientAuth:   tls.RequireAndVerifyClientCert,
				})
				opts = append(opts, grpc.Creds(tlsCreds))
				return opts, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "localhost:0")
			require.NoError(t, err)

			scheme := "https"
			if tt.args.otlpConfig.Tls != nil && tt.args.otlpConfig.Tls.Insecure != nil && *tt.args.otlpConfig.Tls.Insecure {
				scheme = "http"
			}
			tt.args.otlpConfig.Endpoint = new(scheme + "://" + n.Addr().String())

			serverOpts, err := tt.grpcServerOpts()
			require.NoError(t, err)

			startGRPCMetricCollector(t, n, serverOpts)

			exporter, err := otlpGRPCMetricExporter(tt.args.ctx, tt.args.otlpConfig)
			require.NoError(t, err)

			res, err := resource.New(t.Context())
			require.NoError(t, err)

			assert.NoError(t, exporter.Export(t.Context(), &metricdata.ResourceMetrics{
				Resource: res,
				ScopeMetrics: []metricdata.ScopeMetrics{
					{
						Metrics: []metricdata.Metrics{
							{
								Name: "test-metric",
								Data: metricdata.Gauge[int64]{
									DataPoints: []metricdata.DataPoint[int64]{
										{
											Value: 1,
										},
									},
								},
							},
						},
					},
				},
			}))
		})
	}
}

// grpcMetricCollector is an OTLP gRPC server that collects all requests it receives.
type grpcMetricCollector struct {
	v1.UnimplementedMetricsServiceServer
}

var _ v1.MetricsServiceServer = (*grpcMetricCollector)(nil)

// startGRPCMetricCollector returns a *grpcMetricCollector that is listening at the provided
// endpoint.
//
// If endpoint is an empty string, the returned collector will be listening on
// the localhost interface at an OS chosen port.
func startGRPCMetricCollector(t *testing.T, listener net.Listener, serverOptions []grpc.ServerOption) {
	srv := grpc.NewServer(serverOptions...)
	c := &grpcMetricCollector{}

	v1.RegisterMetricsServiceServer(srv, c)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(listener) }()

	t.Cleanup(func() {
		srv.Stop()
		if err := <-errCh; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			assert.NoError(t, err)
		}
	})
}

// Export handles the export req.
func (*grpcMetricCollector) Export(
	_ context.Context,
	_ *v1.ExportMetricsServiceRequest,
) (*v1.ExportMetricsServiceResponse, error) {
	return &v1.ExportMetricsServiceResponse{}, nil
}
