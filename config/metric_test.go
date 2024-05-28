// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
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
									Exporter: MetricExporter{
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
		wantErr    error
		wantReader sdkmetric.Reader
	}{
		{
			name:    "no reader",
			wantErr: errors.New("no valid metric reader"),
		},
		{
			name: "pull/no-exporter",
			reader: MetricReader{
				Pull: &PullMetricReader{},
			},
			wantErr: errors.New("no valid metric exporter"),
		},
		{
			name: "pull/prometheus-no-host",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{},
					},
				},
			},
			wantErr: errors.New("host must be specified"),
		},
		{
			name: "pull/prometheus-no-port",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: ptr("localhost"),
						},
					},
				},
			},
			wantErr: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: ptr("localhost"),
							Port: ptr(8888),
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol: "http/invalid",
						},
					},
				},
			},
			wantErr: errors.New("unsupported protocol \"http/invalid\""),
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4318",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    " ",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: " ", Err: errors.New("invalid URI for request")},
		},
		{
			name: "periodic/grpc-http-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPCExporter),
		},
		{
			name: "periodic/otlp-grpc-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("invalid"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/otlp-http-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-exporter-with-path",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    " ",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: " ", Err: errors.New("invalid URI for request")},
		},
		{
			name: "periodic/otlp-http-none-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTPExporter),
		},
		{
			name: "periodic/otlp-http-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("invalid"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
		{
			name: "periodic/no-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{},
				},
			},
			wantErr: errors.New("no valid metric exporter"),
		},
		{
			name: "periodic/console-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
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
					Interval: newIntPtr(30_000),
					Timeout:  newIntPtr(5_000),
					Exporter: MetricExporter{
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
			require.Equal(t, tt.wantErr, err)
			if tt.wantReader == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantReader), reflect.TypeOf(got))
				var fieldName string
				switch reflect.TypeOf(tt.wantReader).String() {
				case "*metric.PeriodicReader":
					fieldName = "exporter"
				case "config.readerWithServer":
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

func newIntPtr(i int) *int {
	return &i
}
