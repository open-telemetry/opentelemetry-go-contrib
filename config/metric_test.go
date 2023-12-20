// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"

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

func TestInitMeterProvider(t *testing.T) {
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
			name: "reader-and-views",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					MeterProvider: &MeterProvider{
						Readers: []MetricReader{
							{
								Periodic: &PeriodicMetricReader{
									Exporter: MetricExporter{
										Console: Console{},
									},
								},
							},
						},
						Views: []View{
							{
								Selector: &ViewSelector{
									InstrumentName: strPtr("instrument-name"),
								},
							},
						},
					},
				},
			},
			wantProvider: sdkmetric.NewMeterProvider(),
		},
	}
	for _, tt := range tests {
		got, shutdown, err := initMeterProvider(tt.cfg, resource.Default())
		require.Equal(t, reflect.TypeOf(tt.wantProvider), reflect.TypeOf(got))
		require.NoError(t, tt.wantErr, err)
		require.NoError(t, shutdown(context.Background()))
	}
}

func TestPeriodicMetricReader(t *testing.T) {
	console, _ := stdoutmetric.New()
	otlpGRPC, _ := otlpmetricgrpc.New(context.TODO())
	otlpHTTP, _ := otlpmetrichttp.New(context.TODO())
	testCases := []struct {
		name       string
		reader     MetricReader
		args       any
		wantErr    error
		wantReader sdkmetric.Reader
	}{
		{
			name:    "noreader",
			wantErr: errors.New("unsupported metric reader type {<nil> <nil>}"),
		},
		{
			name: "periodic/invalid-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: strPtr("locahost"),
							Port: intPtr(8080),
						},
					},
				},
			},
			wantErr: errNoValidMetricExporter,
		},
		{
			name: "periodic/no-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{},
			},
			wantErr: errNoValidMetricExporter,
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
			wantReader: sdkmetric.NewPeriodicReader(console),
		},
		{
			name: "periodic/console-exporter-with-timeout-interval",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Interval: intPtr(10),
					Timeout:  intPtr(5),
					Exporter: MetricExporter{
						Console: Console{},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(console),
		},
		{
			name: "periodic/otlp-exporter-invalid-protocol",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol: *strPtr("http/invalid"),
						},
					},
				},
			},
			wantErr: errors.New("unsupported protocol http/invalid"),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPC),
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPC),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-scheme",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpGRPC),
		},
		{
			name: "periodic/otlp-grpc-invalid-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "periodic/otlp-grpc-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
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
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTP),
		},
		{
			name: "periodic/otlp-http-exporter-with-path",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTP),
		},
		{
			name: "periodic/otlp-http-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTP),
		},
		{
			name: "periodic/otlp-http-exporter-no-scheme",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantReader: sdkmetric.NewPeriodicReader(otlpHTTP),
		},
		{
			name: "periodic/otlp-http-invalid-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "periodic/otlp-http-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initMetricReader(context.Background(), tt.reader)
			require.Equal(t, tt.wantErr, err)
			if tt.wantReader == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantReader), reflect.TypeOf(got))
				gotPeriodicReader, ok := got.(*sdkmetric.PeriodicReader)
				require.True(t, ok)
				require.NotNil(t, gotPeriodicReader)
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantReader)).FieldByName("exporter").Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
			}
		})
	}
}

func TestPullMetricReader(t *testing.T) {
	prometheusReader, err := otelprom.New()
	require.NoError(t, err)
	testCases := []struct {
		name       string
		reader     MetricReader
		args       any
		wantErr    error
		wantReader sdkmetric.Reader
	}{
		{
			name: "pull prometheus invalid exporter",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{},
					},
				},
			},
			wantErr: errNoValidMetricExporter,
		},
		{
			name: "pull/prometheus-invalid-config-no-host",
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
			name: "pull/prometheus-invalid-config-no-port",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: strPtr("locahost"),
						},
					},
				},
			},
			wantErr: errors.New("port must be specified"),
		},
		{
			name: "pull/prometheus-valid-config",
			reader: MetricReader{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: strPtr("locahost"),
							Port: intPtr(8080),
						},
					},
				},
			},
			wantReader: prometheusReader,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initMetricReader(context.Background(), tt.reader)
			assert.Equal(t, tt.wantErr, err)
			if tt.wantReader == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantReader), reflect.TypeOf(got))
			}
		})
	}
}

func TestInitView(t *testing.T) {
	testCases := []struct {
		name     string
		view     View
		args     any
		wantErr  error
		wantView sdkmetric.View
	}{
		{
			name:     "nil view",
			wantView: sdkmetric.NewView(sdkmetric.Instrument{}, sdkmetric.Stream{}),
		},
		{
			name: "view match instrument no stream",
			view: View{
				Selector: &ViewSelector{
					InstrumentName: strPtr("view-instrument"),
				},
			},
			wantView: sdkmetric.NewView(sdkmetric.Instrument{}, sdkmetric.Stream{}),
		},
		{
			name: "view match instrument no stream",
			view: View{
				Selector: &ViewSelector{
					InstrumentName: strPtr("view-instrument"),
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("counter")),
					MeterName:      strPtr("view-meter"),
					MeterSchemaUrl: strPtr("https://schema.otel.io/1234"),
					Unit:           strPtr("s"),
				},
			},
			wantView: sdkmetric.NewView(sdkmetric.Instrument{}, sdkmetric.Stream{}),
		},
		{
			name: "view instrument observable gauge",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("observable_gauge")),
				},
			},
		},
		{
			name: "view instrument observable counter",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("observable_counter")),
				},
			},
		},
		{
			name: "view instrument observable up down counter",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("observable_up_down_counter")),
				},
			},
		},
		{
			name: "view instrument up down counter",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("up_down_counter")),
				},
			},
		},
		{
			name: "view instrument histogram",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("histogram")),
				},
			},
		},
		{
			name: "view invalid instrument",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("other")),
				},
			},
			wantErr: errors.New("invalid instrument type: other"),
		},
		{
			name: "view instrument with stream",
			view: View{
				Selector: &ViewSelector{
					InstrumentType: (*ViewSelectorInstrumentType)(strPtr("counter")),
				},
				Stream: &ViewStream{
					Description: strPtr("stream-description"),
					Name:        strPtr("stream-name"),
					Aggregation: &ViewStreamAggregation{
						LastValue: ViewStreamAggregationLastValue{},
					},
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initView(tt.view)
			require.Equal(t, tt.wantErr, err)
			require.NotNil(t, got)
			// shouldMatch :=
			// stream, matched := got()
			// if tt.wantView == nil {
			// 	require.Nil(t, got)
			// } else {
			// 	// require.Equal(t, reflect.TypeOf(tt.wantView), reflect.TypeOf(got))
			// 	require.EqualExportedValues(t, tt.wantView, got)
			// }
		})
	}
}
