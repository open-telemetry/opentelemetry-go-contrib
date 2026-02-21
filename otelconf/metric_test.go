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
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
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
							Endpoint: ptr(srv.URL),
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
	consoleExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := t.Context()
	otlpGRPCExporter, err := otlpmetricgrpc.New(ctx)
	require.NoError(t, err)
	otlpHTTPExporter, err := otlpmetrichttp.New(ctx)
	require.NoError(t, err)
	promExporter, err := otelprom.New()
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
			name: "pull/prometheus-no-host",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("host must be specified"),
		},
		{
			name: "pull/prometheus-no-port",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{
							Host: ptr("localhost"),
						},
					},
				},
			},
			wantErrT: newErrInvalid("port must be specified"),
		},
		{
			name: "pull/prometheus",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{
							Host:                ptr("localhost"),
							Port:                ptr(0),
							WithoutScopeInfo:    ptr(true),
							TranslationStrategy: ptr(ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithoutSuffixes),
							WithResourceConstantLabels: &IncludeExclude{
								Included: []string{"include"},
								Excluded: []string{"exclude"},
							},
						},
					},
				},
			},
			wantReader: readerWithServer{promExporter, nil},
		},
		{
			name: "pull/prometheus/invalid strategy",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{
							Host:                ptr("localhost"),
							Port:                ptr(0),
							WithoutScopeInfo:    ptr(true),
							TranslationStrategy: ptr(ExperimentalPrometheusTranslationStrategy("invalid-strategy")),
							WithResourceConstantLabels: &IncludeExclude{
								Included: []string{"include"},
								Excluded: []string{"exclude"},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("translation strategy invalid"),
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							Endpoint:    ptr("http://localhost:4318"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("http://localhost:4318/path/123"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(filepath.Join("testdata", "ca.crt")),
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
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
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
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								KeyFile:  ptr(filepath.Join("testdata", "bad_cert.crt")),
								CertFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
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
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
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
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("unix:collector.sock"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr(" "),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceDelta),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceCumulative),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceLowMemory),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: (*ExporterTemporalityPreference)(ptr("invalid")),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("invalid"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("http://localhost:4318"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(filepath.Join("testdata", "ca.crt")),
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
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
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
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								KeyFile:  ptr(filepath.Join("testdata", "bad_cert.crt")),
								CertFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
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
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
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
							Endpoint:    ptr("http://localhost:4318/path/123"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr(" "),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceCumulative),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceLowMemory),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr(ExporterTemporalityPreferenceDelta),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: (*ExporterTemporalityPreference)(ptr("invalid")),
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
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("invalid"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported compression \"invalid\""),
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
					Interval: ptr(30_000),
					Timeout:  ptr(5_000),
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
		{
			name: "periodic/otlp_file",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileMetricExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("otlp_file/development"),
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
					InstrumentType: (*InstrumentType)(ptr("invalid_type")),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: (*InstrumentType)(ptr("histogram")),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					InstrumentType: ptr(InstrumentTypeCounter),
					Unit:           ptr("test_unit"),
					MeterName:      ptr("test_meter_name"),
					MeterVersion:   ptr("test_meter_version"),
					MeterSchemaUrl: ptr("test_schema_url"),
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
					InstrumentName: ptr("test_name"),
					Unit:           ptr("test_unit"),
				},
				Stream: ViewStream{
					Name:          ptr("new_name"),
					Description:   ptr("new_description"),
					AttributeKeys: ptr(IncludeExclude{Included: []string{"foo", "bar"}}),
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
			instType: ptr(InstrumentTypeCounter),
			wantKind: sdkmetric.InstrumentKindCounter,
		},
		{
			name:     "up_down_counter",
			instType: ptr(InstrumentTypeUpDownCounter),
			wantKind: sdkmetric.InstrumentKindUpDownCounter,
		},
		{
			name:     "histogram",
			instType: ptr(InstrumentTypeHistogram),
			wantKind: sdkmetric.InstrumentKindHistogram,
		},
		{
			name:     "observable_counter",
			instType: ptr(InstrumentTypeObservableCounter),
			wantKind: sdkmetric.InstrumentKindObservableCounter,
		},
		{
			name:     "observable_up_down_counter",
			instType: ptr(InstrumentTypeObservableUpDownCounter),
			wantKind: sdkmetric.InstrumentKindObservableUpDownCounter,
		},
		{
			name:     "observable_gauge",
			instType: ptr(InstrumentTypeObservableGauge),
			wantKind: sdkmetric.InstrumentKindObservableGauge,
		},
		{
			name:     "invalid",
			instType: (*InstrumentType)(ptr("invalid")),
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
					MaxSize:      ptr(2),
					MaxScale:     ptr(3),
					RecordMinMax: ptr(true),
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
					RecordMinMax: ptr(true),
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
			attributeKeys: ptr(IncludeExclude{
				Included: []string{"foo"},
			}),
			wantPass: []string{"foo"},
			wantFail: []string{"bar"},
		},
		{
			name: "filter-with-exclude",
			attributeKeys: ptr(IncludeExclude{
				Excluded: []string{"foo"},
			}),
			wantPass: []string{"bar"},
			wantFail: []string{"foo"},
		},
		{
			name: "filter-with-include-and-exclude",
			attributeKeys: ptr(IncludeExclude{
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
	_, err := newIncludeExcludeFilter(ptr(IncludeExclude{
		Included: []string{"foo"},
		Excluded: []string{"foo"},
	}))
	require.Equal(t, fmt.Errorf("attribute cannot be in both include and exclude list: foo"), err)
}

func TestPrometheusReaderOpts(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         ExperimentalPrometheusMetricExporter
		wantOptions int
	}{
		{
			name:        "no options",
			cfg:         ExperimentalPrometheusMetricExporter{},
			wantOptions: 0,
		},
		{
			name: "all set",
			cfg: ExperimentalPrometheusMetricExporter{
				WithoutScopeInfo:           ptr(true),
				TranslationStrategy:        ptr(ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithoutSuffixes),
				WithResourceConstantLabels: &IncludeExclude{},
			},
			wantOptions: 3,
		},
		{
			name: "all set false",
			cfg: ExperimentalPrometheusMetricExporter{
				WithoutScopeInfo:           ptr(false),
				TranslationStrategy:        ptr(ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithSuffixes),
				WithResourceConstantLabels: &IncludeExclude{},
			},
			wantOptions: 2,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := prometheusReaderOpts(&tt.cfg)
			require.NoError(t, err)
			require.Len(t, opts, tt.wantOptions)
		})
	}
}

func TestPrometheusIPv6(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{
			name: "IPv6",
			host: "::1",
		},
		{
			name: "[IPv6]",
			host: "[::1]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 0
			cfg := ExperimentalPrometheusMetricExporter{
				Host:                       &tt.host,
				Port:                       &port,
				WithoutScopeInfo:           ptr(true),
				TranslationStrategy:        ptr(ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithSuffixes),
				WithResourceConstantLabels: &IncludeExclude{},
			}

			rs, err := prometheusReader(t.Context(), &cfg)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				require.NoError(t, rs.Shutdown(context.Background()))
			})
			require.NoError(t, err)

			hServ := rs.(readerWithServer).server
			assert.True(t, strings.HasPrefix(hServ.Addr, "[::1]:"))

			resp, err := http.DefaultClient.Get("http://" + hServ.Addr + "/metrics")
			t.Cleanup(func() {
				require.NoError(t, resp.Body.Close())
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func Test_otlpGRPCMetricExporter(t *testing.T) {
	if runtime.GOOS == "windows" {
		// TODO (#7446): Fix the flakiness on Windows.
		t.Skip("Test is flaky on Windows.")
	}
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
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Tls: &GrpcTls{
						Insecure: ptr(true),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
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
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Tls: &GrpcTls{
						CaFile: ptr("testdata/server-certs/server.crt"),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				tlsCreds, err := credentials.NewServerTLSFromFile("testdata/server-certs/server.crt", "testdata/server-certs/server.key")
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
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Tls: &GrpcTls{
						CaFile:   ptr("testdata/server-certs/server.crt"),
						KeyFile:  ptr("testdata/client-certs/client.key"),
						CertFile: ptr("testdata/client-certs/client.crt"),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				cert, err := tls.LoadX509KeyPair("testdata/server-certs/server.crt", "testdata/server-certs/server.key")
				if err != nil {
					return nil, err
				}
				caCert, err := os.ReadFile("testdata/ca.crt")
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
			n, err := net.Listen("tcp4", "localhost:0")
			require.NoError(t, err)

			// We need to manually construct the endpoint using the port on which the server is listening.
			//
			// n.Addr() always returns 127.0.0.1 instead of localhost.
			// But our certificate is created with CN as 'localhost', not '127.0.0.1'.
			// So we have to manually form the endpoint as "localhost:<port>".
			_, port, err := net.SplitHostPort(n.Addr().String())
			require.NoError(t, err)
			tt.args.otlpConfig.Endpoint = ptr("localhost:" + port)

			serverOpts, err := tt.grpcServerOpts()
			require.NoError(t, err)

			startGRPCMetricCollector(t, n, serverOpts)

			exporter, err := otlpGRPCMetricExporter(tt.args.ctx, tt.args.otlpConfig)
			require.NoError(t, err)

			res, err := resource.New(t.Context())
			require.NoError(t, err)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				assert.NoError(collect, exporter.Export(context.Background(), &metricdata.ResourceMetrics{ //nolint:usetesting // required to avoid getting a canceled context.
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
			}, 10*time.Second, 1*time.Second)
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

	// Wait for the gRPC server to start accepting connections
	// to avoid race-related test flakiness.
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond)
		if !assert.NoError(collect, err, "failed to dial gRPC server") {
			return
		}
		_ = conn.Close()
	}, 10*time.Second, 1*time.Second)

	t.Cleanup(func() {
		srv.GracefulStop()
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
