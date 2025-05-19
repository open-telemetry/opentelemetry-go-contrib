// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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
			wantErr:      errors.Join(errors.New("must not specify multiple metric reader type")),
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
										Console: Console{},
										OTLP:    &OTLPMetric{},
									},
								},
							},
						},
					},
				},
			},
			wantProvider: noop.NewMeterProvider(),
			wantErr:      errors.Join(errors.New("must not specify multiple metric reader type"), errors.New("must not specify multiple exporters")),
		},
	}
	for _, tt := range tests {
		mp, shutdown, err := meterProvider(tt.cfg, resource.Default())
		require.Equal(t, tt.wantProvider, mp)
		assert.Equal(t, tt.wantErr, err)
		require.NoError(t, shutdown(context.Background()))
	}
}

func TestReader(t *testing.T) {
	consoleExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := context.Background()
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
		wantErr    string
		wantReader sdkmetric.Reader
	}{
		{
			name:    "no reader",
			wantErr: "no valid metric reader",
		},
		{
			name: "pull/no-exporter",
			reader: MetricReader{
				Pull: &PullMetricReader{},
			},
			wantErr: "no valid metric exporter",
		},
		{
			name: "pull/prometheus-no-host",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						Prometheus: &Prometheus{},
					},
				},
			},
			wantErr: "host must be specified",
		},
		{
			name: "pull/prometheus-no-port",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						Prometheus: &Prometheus{
							Host: ptr("localhost"),
						},
					},
				},
			},
			wantErr: "port must be specified",
		},
		{
			name: "pull/prometheus",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						Prometheus: &Prometheus{
							Host:              ptr("localhost"),
							Port:              ptr(0),
							WithoutScopeInfo:  ptr(true),
							WithoutUnits:      ptr(true),
							WithoutTypeSuffix: ptr(true),
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
			name: "periodic/otlp-exporter-invalid-protocol",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol: ptr("http/invalid"),
						},
					},
				},
			},
			wantErr: "unsupported protocol \"http/invalid\"",
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "ca.crt")),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not create certificate authority chain from certificate",
		},
		{
			name: "periodic/otlp-grpc-bad-client-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:          ptr("grpc"),
							Endpoint:          ptr("localhost:4317"),
							Compression:       ptr("gzip"),
							Timeout:           ptr(1000),
							ClientCertificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
							ClientKey:         ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not use client certificate: tls: failed to find any PEM data in certificate input",
		},
		{
			name: "periodic/otlp-grpc-bad-headerslist",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErr: "invalid headers list: invalid key: \"\"",
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
			wantErr: "parse \" \": invalid URI for request",
		},
		{
			name: "periodic/otlp-grpc-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("delta"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("cumulative"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("lowmemory"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("invalid"),
						},
					},
				},
			},
			wantErr: "unsupported temporality preference \"invalid\"",
		},
		{
			name: "periodic/otlp-grpc-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("grpc"),
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
			wantErr: "unsupported compression \"invalid\"",
		},
		{
			name: "periodic/otlp-http-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "ca.crt")),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("https://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not create certificate authority chain from certificate",
		},
		{
			name: "periodic/otlp-http-bad-client-certificate",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:          ptr("http/protobuf"),
							Endpoint:          ptr("localhost:4317"),
							Compression:       ptr("gzip"),
							Timeout:           ptr(1000),
							ClientCertificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
							ClientKey:         ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not use client certificate: tls: failed to find any PEM data in certificate input",
		},
		{
			name: "periodic/otlp-http-bad-headerslist",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErr: "invalid headers list: invalid key: \"\"",
		},
		{
			name: "periodic/otlp-http-exporter-with-path",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
			wantErr: "parse \" \": invalid URI for request",
		},
		{
			name: "periodic/otlp-http-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("cumulative"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("lowmemory"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("delta"),
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
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4318"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
							TemporalityPreference: ptr("invalid"),
						},
					},
				},
			},
			wantErr: "unsupported temporality preference \"invalid\"",
		},
		{
			name: "periodic/otlp-http-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    ptr("http/protobuf"),
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
			wantErr: "unsupported compression \"invalid\"",
		},
		{
			name: "periodic/no-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{},
				},
			},
			wantErr: "no valid metric exporter",
		},
		{
			name: "periodic/console-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						Console: Console{},
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
						Console: Console{},
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
			got, err := metricReader(context.Background(), tt.reader)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
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
				require.NoError(t, got.Shutdown(context.Background()))
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
			name:    "no selector",
			wantErr: "view: no selector provided",
		},
		{
			name: "selector/invalid_type",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("invalid_type")),
				},
			},
			wantErr: "view_selector: instrument_type: invalid value",
		},
		{
			name: "selector/invalid_type",
			view: View{
				Selector: &ViewSelector{},
			},
			wantErr: "view_selector: empty selector not supporter",
		},
		{
			name: "all selectors match",
			view: View{
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("histogram")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					InstrumentType: (*ViewSelectorInstrumentType)(ptr("counter")),
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
				Selector: &ViewSelector{
					InstrumentName: ptr("test_name"),
					Unit:           ptr("test_unit"),
				},
				Stream: &ViewStream{
					Name:          ptr("new_name"),
					Description:   ptr("new_description"),
					AttributeKeys: ptr(IncludeExclude{Included: []string{"foo", "bar"}}),
					Aggregation:   &ViewStreamAggregation{Sum: make(ViewStreamAggregationSum)},
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
		instType *ViewSelectorInstrumentType
		wantErr  error
		wantKind sdkmetric.InstrumentKind
	}{
		{
			name:     "nil",
			wantKind: sdkmetric.InstrumentKind(0),
		},
		{
			name:     "counter",
			instType: (*ViewSelectorInstrumentType)(ptr("counter")),
			wantKind: sdkmetric.InstrumentKindCounter,
		},
		{
			name:     "up_down_counter",
			instType: (*ViewSelectorInstrumentType)(ptr("up_down_counter")),
			wantKind: sdkmetric.InstrumentKindUpDownCounter,
		},
		{
			name:     "histogram",
			instType: (*ViewSelectorInstrumentType)(ptr("histogram")),
			wantKind: sdkmetric.InstrumentKindHistogram,
		},
		{
			name:     "observable_counter",
			instType: (*ViewSelectorInstrumentType)(ptr("observable_counter")),
			wantKind: sdkmetric.InstrumentKindObservableCounter,
		},
		{
			name:     "observable_up_down_counter",
			instType: (*ViewSelectorInstrumentType)(ptr("observable_up_down_counter")),
			wantKind: sdkmetric.InstrumentKindObservableUpDownCounter,
		},
		{
			name:     "observable_gauge",
			instType: (*ViewSelectorInstrumentType)(ptr("observable_gauge")),
			wantKind: sdkmetric.InstrumentKindObservableGauge,
		},
		{
			name:     "invalid",
			instType: (*ViewSelectorInstrumentType)(ptr("invalid")),
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
		aggregation     *ViewStreamAggregation
		wantAggregation sdkmetric.Aggregation
	}{
		{
			name:            "nil",
			wantAggregation: nil,
		},
		{
			name:            "empty",
			aggregation:     &ViewStreamAggregation{},
			wantAggregation: nil,
		},
		{
			name: "Base2ExponentialBucketHistogram empty",
			aggregation: &ViewStreamAggregation{
				Base2ExponentialBucketHistogram: &ViewStreamAggregationBase2ExponentialBucketHistogram{},
			},
			wantAggregation: sdkmetric.AggregationBase2ExponentialHistogram{
				MaxSize:  0,
				MaxScale: 0,
				NoMinMax: true,
			},
		},
		{
			name: "Base2ExponentialBucketHistogram",
			aggregation: &ViewStreamAggregation{
				Base2ExponentialBucketHistogram: &ViewStreamAggregationBase2ExponentialBucketHistogram{
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
			aggregation: &ViewStreamAggregation{
				Default: make(ViewStreamAggregationDefault),
			},
			wantAggregation: nil,
		},
		{
			name: "Drop",
			aggregation: &ViewStreamAggregation{
				Drop: make(ViewStreamAggregationDrop),
			},
			wantAggregation: sdkmetric.AggregationDrop{},
		},
		{
			name: "ExplicitBucketHistogram empty",
			aggregation: &ViewStreamAggregation{
				ExplicitBucketHistogram: &ViewStreamAggregationExplicitBucketHistogram{},
			},
			wantAggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: nil,
				NoMinMax:   true,
			},
		},
		{
			name: "ExplicitBucketHistogram",
			aggregation: &ViewStreamAggregation{
				ExplicitBucketHistogram: &ViewStreamAggregationExplicitBucketHistogram{
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
			aggregation: &ViewStreamAggregation{
				LastValue: make(ViewStreamAggregationLastValue),
			},
			wantAggregation: sdkmetric.AggregationLastValue{},
		},
		{
			name: "Sum",
			aggregation: &ViewStreamAggregation{
				Sum: make(ViewStreamAggregationSum),
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
		cfg         Prometheus
		wantOptions int
	}{
		{
			name:        "no options",
			cfg:         Prometheus{},
			wantOptions: 0,
		},
		{
			name: "all set",
			cfg: Prometheus{
				WithoutScopeInfo:           ptr(true),
				WithoutTypeSuffix:          ptr(true),
				WithoutUnits:               ptr(true),
				WithResourceConstantLabels: &IncludeExclude{},
			},
			wantOptions: 4,
		},
		{
			name: "all set false",
			cfg: Prometheus{
				WithoutScopeInfo:           ptr(false),
				WithoutTypeSuffix:          ptr(false),
				WithoutUnits:               ptr(false),
				WithResourceConstantLabels: &IncludeExclude{},
			},
			wantOptions: 1,
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
			cfg := Prometheus{
				Host:                       &tt.host,
				Port:                       &port,
				WithoutScopeInfo:           ptr(true),
				WithoutTypeSuffix:          ptr(true),
				WithoutUnits:               ptr(true),
				WithResourceConstantLabels: &IncludeExclude{},
			}

			rs, err := prometheusReader(context.Background(), &cfg)
			t.Cleanup(func() {
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
	type args struct {
		ctx        context.Context
		otlpConfig *OTLPMetric
	}
	tests := []struct {
		name           string
		args           args
		grpcServerOpts func() ([]grpc.ServerOption, error)
	}{
		{
			name: "no TLS config",
			args: args{
				ctx: context.Background(),
				otlpConfig: &OTLPMetric{
					Protocol:    ptr("grpc"),
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Insecure:    ptr(true),
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
				ctx: context.Background(),
				otlpConfig: &OTLPMetric{
					Protocol:    ptr("grpc"),
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Certificate: ptr("testdata/server-certs/server.crt"),
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
				ctx: context.Background(),
				otlpConfig: &OTLPMetric{
					Protocol:          ptr("grpc"),
					Compression:       ptr("gzip"),
					Timeout:           ptr(5000),
					Certificate:       ptr("testdata/server-certs/server.crt"),
					ClientKey:         ptr("testdata/client-certs/client.key"),
					ClientCertificate: ptr("testdata/client-certs/client.crt"),
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
			n, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)

			// this is a workaround, as providing 127.0.0.1 resulted in an "invalid URI for request" error
			tt.args.otlpConfig.Endpoint = ptr(strings.ReplaceAll(n.Addr().String(), "127.0.0.1", "localhost"))

			serverOpts, err := tt.grpcServerOpts()
			require.NoError(t, err)

			startGRPCMetricCollector(t, n, serverOpts)

			exporter, err := otlpGRPCMetricExporter(tt.args.ctx, tt.args.otlpConfig)
			require.NoError(t, err)

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				assert.NoError(collect, exporter.Export(context.Background(), &metricdata.ResourceMetrics{
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
	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() {
		srv.Stop()
	})
}

// Export handles the export req.
func (c *grpcMetricCollector) Export(
	_ context.Context,
	_ *v1.ExportMetricsServiceRequest,
) (*v1.ExportMetricsServiceResponse, error) {
	return &v1.ExportMetricsServiceResponse{}, nil
}
