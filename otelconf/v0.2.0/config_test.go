// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewSDK(t *testing.T) {
	tests := []struct {
		name               string
		cfg                []ConfigurationOption
		wantTracerProvider any
		wantMeterProvider  any
		wantLoggerProvider any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "no-configuration",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
		{
			name: "with-configuration",
			cfg: []ConfigurationOption{
				WithContext(t.Context()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{},
					MeterProvider:  &MeterProvider{},
					LoggerProvider: &LoggerProvider{},
				}),
			},
			wantTracerProvider: &sdktrace.TracerProvider{},
			wantMeterProvider:  &sdkmetric.MeterProvider{},
			wantLoggerProvider: &sdklog.LoggerProvider{},
		},
		{
			name: "with-sdk-disabled",
			cfg: []ConfigurationOption{
				WithContext(t.Context()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					Disabled:       new(true),
					TracerProvider: &TracerProvider{},
					MeterProvider:  &MeterProvider{},
					LoggerProvider: &LoggerProvider{},
				}),
			},
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
	}
	for _, tt := range tests {
		sdk, err := NewSDK(tt.cfg...)
		require.Equal(t, tt.wantErr, err)
		assert.IsType(t, tt.wantTracerProvider, sdk.TracerProvider())
		assert.IsType(t, tt.wantMeterProvider, sdk.MeterProvider())
		assert.IsType(t, tt.wantLoggerProvider, sdk.LoggerProvider())
		require.Equal(t, tt.wantShutdownErr, sdk.Shutdown(t.Context()))
	}
}

var v02OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   new(false),
	FileFormat: "0.2",
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       new(128),
		AttributeValueLengthLimit: new(4096),
	},
	LoggerProvider: &LoggerProvider{
		Limits: &LogRecordLimits{
			AttributeCountLimit:       new(128),
			AttributeValueLengthLimit: new(4096),
		},
		Processors: []LogRecordProcessor{
			{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: new(30000),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Certificate:       new("/app/cert.pem"),
							ClientCertificate: new("/app/cert.pem"),
							ClientKey:         new("/app/cert.pem"),
							Compression:       new("gzip"),
							Endpoint:          "http://localhost:4318",
							Headers: Headers{
								"api-key": "1234",
							},
							Insecure: new(false),
							Protocol: "http/protobuf",
							Timeout:  new(10000),
						},
					},
					MaxExportBatchSize: new(512),
					MaxQueueSize:       new(2048),
					ScheduleDelay:      new(5000),
				},
			},
			{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console: Console{},
					},
				},
			},
		},
	},
	MeterProvider: &MeterProvider{
		Readers: []MetricReader{
			{
				Pull: &PullMetricReader{
					Exporter: MetricExporter{
						Prometheus: &Prometheus{
							Host: new("localhost"),
							Port: new(9464),
							WithResourceConstantLabels: &IncludeExclude{
								Excluded: []string{"service.attr1"},
								Included: []string{"service*"},
							},
							WithoutScopeInfo:  new(false),
							WithoutTypeSuffix: new(false),
							WithoutUnits:      new(false),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						OTLP: &OTLPMetric{
							Certificate:                 new("/app/cert.pem"),
							ClientCertificate:           new("/app/cert.pem"),
							ClientKey:                   new("/app/cert.pem"),
							Compression:                 new("gzip"),
							DefaultHistogramAggregation: new(OTLPMetricDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    "http://localhost:4318",
							Headers: Headers{
								"api-key": "1234",
							},
							Insecure:              new(false),
							Protocol:              "http/protobuf",
							TemporalityPreference: new("delta"),
							Timeout:               new(10000),
						},
					},
					Interval: new(5000),
					Timeout:  new(30000),
				},
			},
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
					InstrumentName: new("my-instrument"),
					InstrumentType: new(ViewSelectorInstrumentTypeHistogram),
					MeterName:      new("my-meter"),
					MeterSchemaUrl: new("https://opentelemetry.io/schemas/1.16.0"),
					MeterVersion:   new("1.0.0"),
					Unit:           new("ms"),
				},
				Stream: &ViewStream{
					Aggregation: &ViewStreamAggregation{
						ExplicitBucketHistogram: &ViewStreamAggregationExplicitBucketHistogram{
							Boundaries:   []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							RecordMinMax: new(true),
						},
					},
					AttributeKeys: []string{"key1", "key2"},
					Description:   new("new_description"),
					Name:          new("new_instrument_name"),
				},
			},
		},
	},
	Propagator: &Propagator{
		Composite: []string{"tracecontext", "baggage", "b3", "b3multi", "jaeger", "xray", "ottrace"},
	},
	Resource: &Resource{
		Attributes: Attributes{
			"service.name": "unknown_service",
		},
		Detectors: &Detectors{
			Attributes: &DetectorsAttributes{
				Excluded: []string{"process.command_args"},
				Included: []string{"process.*"},
			},
		},
		SchemaUrl: new("https://opentelemetry.io/schemas/1.16.0"),
	},
	TracerProvider: &TracerProvider{
		Limits: &SpanLimits{
			AttributeCountLimit:       new(128),
			AttributeValueLengthLimit: new(4096),
			EventCountLimit:           new(128),
			EventAttributeCountLimit:  new(128),
			LinkCountLimit:            new(128),
			LinkAttributeCountLimit:   new(128),
		},
		Processors: []SpanProcessor{
			{
				Batch: &BatchSpanProcessor{
					ExportTimeout: new(30000),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Certificate:       new("/app/cert.pem"),
							ClientCertificate: new("/app/cert.pem"),
							ClientKey:         new("/app/cert.pem"),
							Compression:       new("gzip"),
							Endpoint:          "http://localhost:4318",
							Headers: Headers{
								"api-key": "1234",
							},
							Insecure: new(false),
							Protocol: "http/protobuf",
							Timeout:  new(10000),
						},
					},
					MaxExportBatchSize: new(512),
					MaxQueueSize:       new(2048),
					ScheduleDelay:      new(5000),
				},
			},
			{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						Zipkin: &Zipkin{
							Endpoint: "http://localhost:9411/api/v2/spans",
							Timeout:  new(10000),
						},
					},
				},
			},
			{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
		},
		Sampler: &Sampler{
			ParentBased: &SamplerParentBased{
				LocalParentNotSampled: &Sampler{
					AlwaysOff: SamplerAlwaysOff{},
				},
				LocalParentSampled: &Sampler{
					AlwaysOn: SamplerAlwaysOn{},
				},
				RemoteParentNotSampled: &Sampler{
					AlwaysOff: SamplerAlwaysOff{},
				},
				RemoteParentSampled: &Sampler{
					AlwaysOn: SamplerAlwaysOn{},
				},
				Root: &Sampler{
					TraceIDRatioBased: &SamplerTraceIDRatioBased{
						Ratio: new(0.0001),
					},
				},
			},
		},
	},
}

func TestParseYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType any
	}{
		{
			name:    "valid YAML config",
			input:   `valid_empty.yaml`,
			wantErr: nil,
			wantType: &OpenTelemetryConfiguration{
				Disabled:   new(false),
				FileFormat: "0.1",
			},
		},
		{
			name:  "invalid config",
			input: "invalid_bool.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 2: cannot unmarshal !!str ` + "`notabool`" + ` into bool`),
		},
		{
			name:     "valid v0.2 config",
			input:    "v0.2.yaml",
			wantType: &v02OpenTelemetryConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("..", "testdata", tt.input))
			require.NoError(t, err)

			got, err := ParseYAML(b)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantType, got)
			}
		})
	}
}

func TestSerializeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType any
	}{
		{
			name:    "valid JSON config",
			input:   `valid_empty.json`,
			wantErr: nil,
			wantType: OpenTelemetryConfiguration{
				Disabled:   new(false),
				FileFormat: "0.1",
			},
		},
		{
			name:    "invalid config",
			input:   "invalid_bool.json",
			wantErr: errors.New(`json: cannot unmarshal string into Go struct field Plain.disabled of type bool`),
		},
		{
			name:     "valid v0.2 config",
			input:    "v0.2.json",
			wantType: v02OpenTelemetryConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("..", "testdata", tt.input))
			require.NoError(t, err)

			var got OpenTelemetryConfiguration
			err = json.Unmarshal(b, &got)

			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantType, got)
			}
		})
	}
}
