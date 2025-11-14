// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
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
		require.NoError(t, shutdown(t.Context()))
	}
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
			name: "periodic/otlp-grpc-none-compression",
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
			name: "periodic/otlp-grpc-delta-temporality",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: ptr("invalid"),
						},
					},
				},
			},
			wantErr: errors.New("unsupported temporality preference \"invalid\""),
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
			name: "periodic/otlp-http-cumulative-temporality",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
							TemporalityPreference: ptr("invalid"),
						},
					},
				},
			},
			wantErr: errors.New("unsupported temporality preference \"invalid\""),
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
					Interval: ptr(30_000),
					Timeout:  ptr(5_000),
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
			got, err := metricReader(t.Context(), tt.reader)
			require.Equal(t, tt.wantErr, err)
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
					AttributeKeys: []string{"foo", "bar"},
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

func TestAttributeFilter(t *testing.T) {
	testCases := []struct {
		name          string
		attributeKeys []string
		wantPass      []string
		wantFail      []string
	}{
		{
			name:          "empty",
			attributeKeys: []string{},
			wantPass:      nil,
			wantFail:      []string{"foo", "bar"},
		},
		{
			name:          "filter",
			attributeKeys: []string{"foo"},
			wantPass:      []string{"foo"},
			wantFail:      []string{"bar"},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := attributeFilter(tt.attributeKeys)
			for _, pass := range tt.wantPass {
				require.True(t, got(attribute.KeyValue{Key: attribute.Key(pass), Value: attribute.StringValue("")}))
			}
			for _, fail := range tt.wantFail {
				require.False(t, got(attribute.KeyValue{Key: attribute.Key(fail), Value: attribute.StringValue("")}))
			}
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

func TestPrometheusReaderErrorCases(t *testing.T) {
	tests := []struct {
		name   string
		config *Prometheus
		errMsg string
	}{
		{
			name:   "missing host",
			config: &Prometheus{Port: ptr(8080)},
			errMsg: "host must be specified",
		},
		{
			name:   "missing port",
			config: &Prometheus{Host: ptr("localhost")},
			errMsg: "port must be specified",
		},
		{
			name: "invalid port",
			config: &Prometheus{
				Host:                       ptr("localhost"),
				Port:                       ptr(99999), // invalid port
				WithoutScopeInfo:           ptr(true),
				WithoutTypeSuffix:          ptr(true),
				WithoutUnits:               ptr(true),
				WithResourceConstantLabels: &IncludeExclude{},
			},
			errMsg: "binding address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := prometheusReader(t.Context(), tt.config)
			assert.ErrorContains(t, err, tt.errMsg)
			assert.Nil(t, reader)
		})
	}
}

func TestPrometheusReaderConfigurationOptions(t *testing.T) {
	host := "localhost"
	port := 0
	cfg := &Prometheus{
		Host:              &host,
		Port:              &port,
		WithoutScopeInfo:  ptr(true),
		WithoutTypeSuffix: ptr(true),
		WithoutUnits:      ptr(true),
		WithResourceConstantLabels: &IncludeExclude{
			Included: []string{"service.name"},
			Excluded: []string{"host.name"},
		},
	}

	reader, err := prometheusReader(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, reader)

	t.Cleanup(func() {
		//nolint:usetesting // required to avoid getting a canceled context at cleanup.
		require.NoError(t, reader.Shutdown(context.Background()))
	})

	rws, ok := reader.(readerWithServer)
	require.True(t, ok, "reader is not a readerWithServer")
	server := rws.server

	addr := server.Addr
	// localhost resolves to 127.0.0.1, so we expect the resolved IP
	assert.Contains(t, addr, "127.0.0.1")

	resp, err := http.Get("http://" + addr + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPrometheusReaderHostParsing(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		wantAddr string
	}{
		{
			name:     "regular host",
			host:     "localhost",
			wantAddr: "127.0.0.1", // localhost resolves to this IP
		},
		{
			name:     "IPv4",
			host:     "127.0.0.1",
			wantAddr: "127.0.0.1",
		},
		{
			name:     "IPv6 with brackets",
			host:     "[::1]",
			wantAddr: "::1",
		},
		{
			name:     "IPv6 without brackets",
			host:     "::1",
			wantAddr: "::1",
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

			reader, err := prometheusReader(t.Context(), &cfg)
			require.NoError(t, err)
			require.NotNil(t, reader)

			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				require.NoError(t, reader.Shutdown(context.Background()))
			})

			rws, ok := reader.(readerWithServer)
			require.True(t, ok, "reader is not a readerWithServer")
			server := rws.server

			assert.Contains(t, server.Addr, tt.wantAddr)
		})
	}
}
