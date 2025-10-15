// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"crypto/tls"
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
					TracerProvider: &TracerProviderJson{},
					MeterProvider:  &MeterProviderJson{},
					LoggerProvider: &LoggerProviderJson{},
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
					Disabled:       ptr(true),
					TracerProvider: &TracerProviderJson{},
					MeterProvider:  &MeterProviderJson{},
					LoggerProvider: &LoggerProviderJson{},
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

var v10OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   ptr(false),
	FileFormat: "1.0-rc.1",
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       ptr(128),
		AttributeValueLengthLimit: ptr(4096),
	},
	InstrumentationDevelopment: &InstrumentationJson{
		Cpp: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Dotnet: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Erlang: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		General: &ExperimentalGeneralInstrumentation{
			Http: &ExperimentalHttpInstrumentation{
				Client: &ExperimentalHttpInstrumentationClient{
					RequestCapturedHeaders:  []string{"Content-Type", "Accept"},
					ResponseCapturedHeaders: []string{"Content-Type", "Content-Encoding"},
				},
				Server: &ExperimentalHttpInstrumentationServer{
					RequestCapturedHeaders:  []string{"Content-Type", "Accept"},
					ResponseCapturedHeaders: []string{"Content-Type", "Content-Encoding"},
				},
			},
			Peer: &ExperimentalPeerInstrumentation{
				ServiceMapping: []ExperimentalPeerInstrumentationServiceMappingElem{
					{Peer: "1.2.3.4", Service: "FooService"},
					{Peer: "2.3.4.5", Service: "BarService"},
				},
			},
		},
		Go: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Java: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Js: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Php: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Python: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Ruby: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Rust: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Swift: ExperimentalLanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
	},
	LogLevel: ptr("info"),
	LoggerProvider: &LoggerProviderJson{
		LoggerConfiguratorDevelopment: &ExperimentalLoggerConfigurator{
			DefaultConfig: &ExperimentalLoggerConfig{
				Disabled: ptr(true),
			},
			Loggers: []ExperimentalLoggerMatcherAndConfig{
				{
					Config: &ExperimentalLoggerConfig{
						Disabled: ptr(false),
					},
					Name: ptr("io.opentelemetry.contrib.*"),
				},
			},
		},
		Limits: &LogRecordLimits{
			AttributeCountLimit:       ptr(128),
			AttributeValueLengthLimit: ptr(4096),
		},
		Processors: []LogRecordProcessor{
			{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: ptr(30000),
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{
							CertificateFile:       ptr("/app/cert.pem"),
							ClientCertificateFile: ptr("/app/cert.pem"),
							ClientKeyFile:         ptr("/app/cert.pem"),
							Compression:           ptr("gzip"),
							Encoding:              ptr(OTLPHttpEncodingProtobuf),
							Endpoint:              ptr("http://localhost:4318/v1/logs"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Timeout:     ptr(10000),
						},
					},
					MaxExportBatchSize: ptr(512),
					MaxQueueSize:       ptr(2048),
					ScheduleDelay:      ptr(5000),
				},
			},
			{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							CertificateFile:       ptr("/app/cert.pem"),
							ClientCertificateFile: ptr("/app/cert.pem"),
							ClientKeyFile:         ptr("/app/cert.pem"),
							Compression:           ptr("gzip"),
							Endpoint:              ptr("http://localhost:4317"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Timeout:     ptr(10000),
							Insecure:    ptr(false),
						},
					},
				},
			},
			{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{
							OutputStream: ptr("file:///var/log/logs.jsonl"),
						},
					},
				},
			},
			{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{
							OutputStream: ptr("stdout"),
						},
					},
				},
			},
			{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console: ConsoleExporter{},
					},
				},
			},
		},
	},
	MeterProvider: &MeterProviderJson{
		ExemplarFilter: ptr(ExemplarFilter("trace_based")),
		MeterConfiguratorDevelopment: &ExperimentalMeterConfigurator{
			DefaultConfig: &ExperimentalMeterConfig{
				Disabled: ptr(true),
			},
			Meters: []ExperimentalMeterMatcherAndConfig{
				{
					Config: &ExperimentalMeterConfig{
						Disabled: ptr(false),
					},
					Name: ptr("io.opentelemetry.contrib.*"),
				},
			},
		},
		Readers: []MetricReader{
			{
				Pull: &PullMetricReader{
					Producers: []MetricProducer{
						{
							Opencensus: OpenCensusMetricProducer{},
						},
					},
					CardinalityLimits: &CardinalityLimits{
						Default:                 ptr(2000),
						Counter:                 ptr(2000),
						Gauge:                   ptr(2000),
						Histogram:               ptr(2000),
						ObservableCounter:       ptr(2000),
						ObservableGauge:         ptr(2000),
						ObservableUpDownCounter: ptr(2000),
						UpDownCounter:           ptr(2000),
					},
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{
							Host: ptr("localhost"),
							Port: ptr(9464),
							WithResourceConstantLabels: &IncludeExclude{
								Excluded: []string{"service.attr1"},
								Included: []string{"service*"},
							},
							WithoutScopeInfo: ptr(false),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Producers: []MetricProducer{
						{
							AdditionalProperties: map[string]any{
								"prometheus": nil,
							},
						},
					},
					CardinalityLimits: &CardinalityLimits{
						Default:                 ptr(2000),
						Counter:                 ptr(2000),
						Gauge:                   ptr(2000),
						Histogram:               ptr(2000),
						ObservableCounter:       ptr(2000),
						ObservableGauge:         ptr(2000),
						ObservableUpDownCounter: ptr(2000),
						UpDownCounter:           ptr(2000),
					},
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							CertificateFile:             ptr("/app/cert.pem"),
							ClientCertificateFile:       ptr("/app/cert.pem"),
							ClientKeyFile:               ptr("/app/cert.pem"),
							Compression:                 ptr("gzip"),
							DefaultHistogramAggregation: ptr(ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    ptr("http://localhost:4318/v1/metrics"),
							Encoding:                    ptr(OTLPHttpEncodingProtobuf),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList:           ptr("api-key=1234"),
							TemporalityPreference: ptr(ExporterTemporalityPreferenceDelta),
							Timeout:               ptr(10000),
						},
					},
					Interval: ptr(60000),
					Timeout:  ptr(30000),
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPGrpc: &OTLPGrpcMetricExporter{
							CertificateFile:             ptr("/app/cert.pem"),
							ClientCertificateFile:       ptr("/app/cert.pem"),
							ClientKeyFile:               ptr("/app/cert.pem"),
							Compression:                 ptr("gzip"),
							DefaultHistogramAggregation: ptr(ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    ptr("http://localhost:4317"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList:           ptr("api-key=1234"),
							TemporalityPreference: ptr(ExporterTemporalityPreferenceDelta),
							Timeout:               ptr(10000),
							Insecure:              ptr(false),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileMetricExporter{
							OutputStream:                ptr("file:///var/log/metrics.jsonl"),
							DefaultHistogramAggregation: ptr(ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							TemporalityPreference:       ptr(ExporterTemporalityPreferenceDelta),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileMetricExporter{
							OutputStream:                ptr("stdout"),
							DefaultHistogramAggregation: ptr(ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							TemporalityPreference:       ptr(ExporterTemporalityPreferenceDelta),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						Console: ConsoleExporter{},
					},
				},
			},
		},
		Views: []View{
			{
				Selector: &ViewSelector{
					InstrumentName: ptr("my-instrument"),
					InstrumentType: ptr(InstrumentTypeHistogram),
					MeterName:      ptr("my-meter"),
					MeterSchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
					MeterVersion:   ptr("1.0.0"),
					Unit:           ptr("ms"),
				},
				Stream: &ViewStream{
					Aggregation: &Aggregation{
						ExplicitBucketHistogram: &ExplicitBucketHistogramAggregation{
							Boundaries:   []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
							RecordMinMax: ptr(true),
						},
					},
					AggregationCardinalityLimit: ptr(2000),
					AttributeKeys: &IncludeExclude{
						Included: []string{"key1", "key2"},
						Excluded: []string{"key3"},
					},
					Description: ptr("new_description"),
					Name:        ptr("new_instrument_name"),
				},
			},
		},
	},
	Propagator: &PropagatorJson{
		Composite: []TextMapPropagator{
			{
				Tracecontext: TraceContextPropagator{},
			},
			{
				Baggage: BaggagePropagator{},
			},
			{
				B3: B3Propagator{},
			},
			{
				B3Multi: B3MultiPropagator{},
			},
			{
				Jaeger: JaegerPropagator{},
			},
			{
				Ottrace: OpenTracingPropagator{},
			},
		},
		CompositeList: ptr("tracecontext,baggage,b3,b3multi,jaeger,ottrace,xray"),
	},
	Resource: &ResourceJson{
		Attributes: []AttributeNameValue{
			{Name: "service.name", Value: "unknown_service"},
			{Name: "string_key", Type: &AttributeType{Value: "string"}, Value: "value"},
			{Name: "bool_key", Type: &AttributeType{Value: "bool"}, Value: true},
			{Name: "int_key", Type: &AttributeType{Value: "int"}, Value: 1},
			{Name: "double_key", Type: &AttributeType{Value: "double"}, Value: 1.1},
			{Name: "string_array_key", Type: &AttributeType{Value: "string_array"}, Value: []any{"value1", "value2"}},
			{Name: "bool_array_key", Type: &AttributeType{Value: "bool_array"}, Value: []any{true, false}},
			{Name: "int_array_key", Type: &AttributeType{Value: "int_array"}, Value: []any{1, 2}},
			{Name: "double_array_key", Type: &AttributeType{Value: "double_array"}, Value: []any{1.1, 2.2}},
		},
		AttributesList: ptr("service.namespace=my-namespace,service.version=1.0.0"),
		DetectionDevelopment: &ExperimentalResourceDetection{
			Attributes: &IncludeExclude{
				Excluded: []string{"process.command_args"},
				Included: []string{"process.*"},
			},
			// TODO: implement resource detectors
			// Detectors: []ExperimentalResourceDetector{}
			// },
		},
		SchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
	},
	TracerProvider: &TracerProviderJson{
		TracerConfiguratorDevelopment: &ExperimentalTracerConfigurator{
			DefaultConfig: &ExperimentalTracerConfig{
				Disabled: true,
			},
			Tracers: []ExperimentalTracerMatcherAndConfig{
				{
					Config: ExperimentalTracerConfig{},
					Name:   "io.opentelemetry.contrib.*",
				},
			},
		},

		Limits: &SpanLimits{
			AttributeCountLimit:       ptr(128),
			AttributeValueLengthLimit: ptr(4096),
			EventCountLimit:           ptr(128),
			EventAttributeCountLimit:  ptr(128),
			LinkCountLimit:            ptr(128),
			LinkAttributeCountLimit:   ptr(128),
		},
		Processors: []SpanProcessor{
			{
				Batch: &BatchSpanProcessor{
					ExportTimeout: ptr(30000),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							CertificateFile:       ptr("/app/cert.pem"),
							ClientCertificateFile: ptr("/app/cert.pem"),
							ClientKeyFile:         ptr("/app/cert.pem"),
							Compression:           ptr("gzip"),
							Encoding:              ptr(OTLPHttpEncodingProtobuf),
							Endpoint:              ptr("http://localhost:4318/v1/traces"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Timeout:     ptr(10000),
						},
					},
					MaxExportBatchSize: ptr(512),
					MaxQueueSize:       ptr(2048),
					ScheduleDelay:      ptr(5000),
				},
			},
			{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							CertificateFile:       ptr("/app/cert.pem"),
							ClientCertificateFile: ptr("/app/cert.pem"),
							ClientKeyFile:         ptr("/app/cert.pem"),
							Compression:           ptr("gzip"),
							Endpoint:              ptr("http://localhost:4317"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Timeout:     ptr(10000),
							Insecure:    ptr(false),
						},
					},
				},
			},
			{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{
							OutputStream: ptr("file:///var/log/traces.jsonl"),
						},
					},
				},
			},
			{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{
							OutputStream: ptr("stdout"),
						},
					},
				},
			},
			{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						Zipkin: &ZipkinSpanExporter{
							Endpoint: ptr("http://localhost:9411/api/v2/spans"),
							Timeout:  ptr(10000),
						},
					},
				},
			},
			{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
		},
		Sampler: &Sampler{
			ParentBased: &ParentBasedSampler{
				LocalParentNotSampled: &Sampler{
					AlwaysOff: AlwaysOffSampler{},
				},
				LocalParentSampled: &Sampler{
					AlwaysOn: AlwaysOnSampler{},
				},
				RemoteParentNotSampled: &Sampler{
					AlwaysOff: AlwaysOffSampler{},
				},
				RemoteParentSampled: &Sampler{
					AlwaysOn: AlwaysOnSampler{},
				},
				Root: &Sampler{
					TraceIDRatioBased: &TraceIDRatioBasedSampler{
						Ratio: ptr(0.0001),
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
				Disabled:   ptr(false),
				FileFormat: "0.1",
				LogLevel:   ptr("info"),
			},
		},
		{
			name:  "invalid config",
			input: "invalid_bool.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 1: cannot unmarshal !!str ` + "`notabool`" + ` into bool`),
		},
		{
			name:    "invalid nil name",
			input:   "v1.0.0_invalid_nil_name.yaml",
			wantErr: errors.New(`yaml: cannot unmarshal field name in NameStringValuePair required`),
		},
		{
			name:    "invalid nil value",
			input:   "v1.0.0_invalid_nil_value.yaml",
			wantErr: errors.New(`yaml: cannot unmarshal field value in NameStringValuePair required`),
		},
		{
			name:  "valid v0.2 config",
			input: "v0.2.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 64: cannot unmarshal !!seq into otelconf.IncludeExclude`),
		},
		{
			name:  "valid v0.3 config",
			input: "v0.3.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 2: cannot unmarshal !!str` + " `traceco...`" + ` into map[string]interface {}
  line 3: cannot unmarshal !!str` + " `baggage`" + ` into map[string]interface {}
  line 4: cannot unmarshal !!str` + " `b3`" + ` into map[string]interface {}
  line 5: cannot unmarshal !!str` + " `b3multi`" + ` into map[string]interface {}
  line 6: cannot unmarshal !!str` + " `jaeger`" + ` into map[string]interface {}
  line 7: cannot unmarshal !!str` + " `xray`" + ` into map[string]interface {}
  line 8: cannot unmarshal !!str` + " `ottrace`" + ` into map[string]interface {}`),
		},
		{
			name:     "valid v1.0.0 config",
			input:    "v1.0.0-rc.1.yaml",
			wantType: &v10OpenTelemetryConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("..", "testdata", tt.input))
			require.NoError(t, err)

			got, err := ParseYAML(b)
			if tt.wantErr != nil {
				require.Error(t, err)
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
				Disabled:   ptr(false),
				FileFormat: "0.1",
				LogLevel:   ptr("info"),
			},
		},
		{
			name:    "invalid config",
			input:   "invalid_bool.json",
			wantErr: errors.New(`json: cannot unmarshal string into Go value of type bool`),
		},
		{
			name:    "invalid nil name",
			input:   "v1.0.0_invalid_nil_name.json",
			wantErr: errors.New(`json: cannot unmarshal field name in NameStringValuePair required`),
		},
		{
			name:    "invalid nil value",
			input:   "v1.0.0_invalid_nil_value.json",
			wantErr: errors.New(`json: cannot unmarshal field value in NameStringValuePair required`),
		},
		{
			name:    "valid v0.2 config",
			input:   "v0.2.json",
			wantErr: errors.New(`json: cannot unmarshal array into Go struct field View.views.stream.attribute_keys of type otelconf.IncludeExclude`),
		},
		{
			name:    "valid v0.3 config",
			input:   "v0.3.json",
			wantErr: errors.New(`json: cannot unmarshal string into Go struct field PropagatorJson.composite of type map[string]interface {}`),
		},
		{
			name:     "valid v1.0.0 config",
			input:    "v1.0.0-rc.1.json",
			wantType: v10OpenTelemetryConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("..", "testdata", tt.input))
			require.NoError(t, err)

			var got OpenTelemetryConfiguration
			err = json.Unmarshal(b, &got)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantType, got)
			}
		})
	}
}

func TestCreateTLSConfig(t *testing.T) {
	tests := []struct {
		name            string
		caCertFile      *string
		clientCertFile  *string
		clientKeyFile   *string
		wantErrContains string
		want            func(*tls.Config, *testing.T)
	}{
		{
			name: "no-input",
			want: func(result *tls.Config, t *testing.T) {
				require.Nil(t, result.Certificates)
				require.Nil(t, result.RootCAs)
			},
		},
		{
			name:       "only-cacert-provided",
			caCertFile: ptr(filepath.Join("..", "testdata", "ca.crt")),
			want: func(result *tls.Config, t *testing.T) {
				require.Nil(t, result.Certificates)
				require.NotNil(t, result.RootCAs)
			},
		},
		{
			name:            "nonexistent-cacert-file",
			caCertFile:      ptr("nowhere.crt"),
			wantErrContains: "open nowhere.crt:",
		},
		{
			name:            "nonexistent-clientcert-file",
			clientCertFile:  ptr("nowhere.crt"),
			clientKeyFile:   ptr("nowhere.crt"),
			wantErrContains: "could not use client certificate: open nowhere.crt:",
		},
		{
			name:            "bad-cacert-file",
			caCertFile:      ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
			wantErrContains: "could not create certificate authority chain from certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createTLSConfig(tt.caCertFile, tt.clientCertFile, tt.clientKeyFile)

			if tt.wantErrContains != "" {
				require.Contains(t, err.Error(), tt.wantErrContains)
			} else {
				require.NoError(t, err)
				tt.want(got, t)
			}
		})
	}
}

func TestCreateHeadersConfig(t *testing.T) {
	tests := []struct {
		name        string
		headers     []NameStringValuePair
		headersList *string
		wantHeaders map[string]string
		wantErr     string
	}{
		{
			name:        "no headers",
			headers:     []NameStringValuePair{},
			headersList: nil,
			wantHeaders: map[string]string{},
		},
		{
			name:        "headerslist only",
			headers:     []NameStringValuePair{},
			headersList: ptr("a=b,c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "headers only",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
				{
					Name:  "c",
					Value: ptr("d"),
				},
			},
			headersList: nil,
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "both headers and headerslist",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
			},
			headersList: ptr("c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "headers supersedes headerslist",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
				{
					Name:  "c",
					Value: ptr("override"),
				},
			},
			headersList: ptr("c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "override",
			},
		},
		{
			name:        "invalid headerslist",
			headersList: ptr("==="),
			wantErr:     "invalid headers list: invalid key: \"\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headersMap, err := createHeadersConfig(tt.headers, tt.headersList)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantHeaders, headersMap)
		})
	}
}
