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
	"go.yaml.in/yaml/v3"

	"github.com/stretchr/testify/require"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestUnmarshalPushMetricExporterInvalidData(t *testing.T) {
	cl := PushMetricExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&PushMetricExporter{}))

	cl = PushMetricExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = PushMetricExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&PushMetricExporter{}))
}

func TestUnmarshalLogRecordExporterInvalidData(t *testing.T) {
	cl := LogRecordExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&LogRecordExporter{}))

	cl = LogRecordExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = LogRecordExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&LogRecordExporter{}))
}

func TestUnmarshalSpanExporterInvalidData(t *testing.T) {
	cl := SpanExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&SpanExporter{}))

	cl = SpanExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = SpanExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&SpanExporter{}))
}

func TestUnmarshalTextMapPropagator(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		yamlConfig            []byte
		jsonConfig            []byte
		wantErrT              error
		wantTextMapPropagator TextMapPropagator
	}{
		{
			name:                  "valid with b3 propagator",
			jsonConfig:            []byte(`{"b3":{}}`),
			yamlConfig:            []byte("b3: {}\n"),
			wantTextMapPropagator: TextMapPropagator{B3: B3Propagator{}},
		},
		{
			name:       "valid with all propagators",
			jsonConfig: []byte(`{"b3":{},"b3multi":{},"baggage":{},"jaeger":{},"ottrace":{},"tracecontext":{}}`),
			yamlConfig: []byte("b3: {}\nb3multi: {}\nbaggage: {}\njaeger: {}\nottrace: {}\ntracecontext: {}\n"),
			wantTextMapPropagator: TextMapPropagator{
				B3:           B3Propagator{},
				B3Multi:      B3MultiPropagator{},
				Baggage:      BaggagePropagator{},
				Jaeger:       JaegerPropagator{},
				Ottrace:      OpenTracingPropagator{},
				Tracecontext: TraceContextPropagator{},
			},
		},
		{
			name:       "valid with all propagators nil",
			jsonConfig: []byte(`{"b3":null,"b3multi":null,"baggage":null,"jaeger":null,"ottrace":null,"tracecontext":null}`),
			yamlConfig: []byte("b3:\nb3multi:\nbaggage:\njaeger:\nottrace:\ntracecontext:\n"),
			wantTextMapPropagator: TextMapPropagator{
				B3:           B3Propagator{},
				B3Multi:      B3MultiPropagator{},
				Baggage:      BaggagePropagator{},
				Jaeger:       JaegerPropagator{},
				Ottrace:      OpenTracingPropagator{},
				Tracecontext: TraceContextPropagator{},
			},
		},
		{
			name:       "invalid b3 data",
			jsonConfig: []byte(`{"b3":2000}`),
			yamlConfig: []byte("b3: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid b3multi data",
			jsonConfig: []byte(`{"b3multi":2000}`),
			yamlConfig: []byte("b3multi: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid baggage data",
			jsonConfig: []byte(`{"baggage":2000}`),
			yamlConfig: []byte("baggage: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid jaeger data",
			jsonConfig: []byte(`{"jaeger":2000}`),
			yamlConfig: []byte("jaeger: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid ottrace data",
			jsonConfig: []byte(`{"ottrace":2000}`),
			yamlConfig: []byte("ottrace: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid tracecontext data",
			jsonConfig: []byte(`{"tracecontext":2000}`),
			yamlConfig: []byte("tracecontext: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := TextMapPropagator{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantTextMapPropagator, cl)

			cl = TextMapPropagator{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantTextMapPropagator, cl)
		})
	}
}

func TestUnmarshalBatchLogRecordProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter LogRecordExporter
	}{
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":5000,"max_export_batch_size":512,"max_queue_size":2048,"schedule_delay":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 5000\nmax_export_batch_size: 512\nmax_queue_size: 2048\nschedule_delay: 1000"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero export_timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 0"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero schedule_delay",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"schedule_delay":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nschedule_delay: 0"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequiredExporter(&BatchLogRecordProcessor{}),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   newErrUnmarshal(&BatchLogRecordProcessor{}),
		},
		{
			name:       "invalid export_timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"export_timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name:       "invalid max_export_batch_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_export_batch_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_queue_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid max_queue_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid schedule_delay negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"schedule_delay":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nschedule_delay: -1"),
			wantErrT:   newErrGreaterOrEqualZero("schedule_delay"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := BatchLogRecordProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = BatchLogRecordProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

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

var v03OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   ptr(false),
	FileFormat: "0.3",
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
	LoggerProvider: &LoggerProviderJson{
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
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console: ConsoleExporter{},
					},
				},
			},
		},
	},
	MeterProvider: &MeterProviderJson{
		Readers: []MetricReader{
			{
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
						PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{
							Host: ptr("localhost"),
							Port: ptr(9464),
							WithResourceConstantLabels: &IncludeExclude{
								Excluded: []string{"service.attr1"},
								Included: []string{"service*"},
							},
							WithoutScopeInfo:  ptr(false),
							WithoutTypeSuffix: ptr(false),
							WithoutUnits:      ptr(false),
						},
					},
				},
			},
			{
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLPHttp: &OTLPHttpMetricExporter{
							CertificateFile:             ptr("/app/cert.pem"),
							ClientCertificateFile:       ptr("/app/cert.pem"),
							ClientKeyFile:               ptr("/app/cert.pem"),
							Compression:                 ptr("gzip"),
							DefaultHistogramAggregation: ptr(ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    ptr("http://localhost:4318/v1/metrics"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList:           ptr("api-key=1234"),
							TemporalityPreference: ptr(ExporterTemporalityPreferenceDelta),
							Timeout:               ptr(10000),
						},
					},
					Interval: ptr(5000),
					Timeout:  ptr(30000),
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
		// Composite: []TextMapPropagator{TraceContextPropagator, ptr("tracecontext"), ptr("baggage"), ptr("b3"), ptr("b3multi"), ptr("jaeger"), ptr("xray"), ptr("ottrace")},
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
		},
		SchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
	},
	TracerProvider: &TracerProviderJson{
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
			},
		},
		{
			name:  "invalid config",
			input: "invalid_bool.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 2: cannot unmarshal !!str ` + "`notabool`" + ` into bool`),
		},
		{
			name:    "invalid nil name",
			input:   "invalid_nil_name.yaml",
			wantErr: errors.New(`yaml: cannot unmarshal field name in NameStringValuePair required`),
		},
		{
			name:    "invalid nil value",
			input:   "invalid_nil_value.yaml",
			wantErr: errors.New(`yaml: cannot unmarshal field value in NameStringValuePair required`),
		},
		{
			name:  "valid v0.2 config",
			input: "v0.2.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 81: cannot unmarshal !!map into []otelconf.NameStringValuePair
  line 185: cannot unmarshal !!map into []otelconf.NameStringValuePair
  line 244: cannot unmarshal !!seq into otelconf.IncludeExclude
  line 305: cannot unmarshal !!map into []otelconf.NameStringValuePair
  line 408: cannot unmarshal !!map into []otelconf.AttributeNameValue`),
		},
		{
			name:     "valid v0.3 config",
			input:    "v0.3.yaml",
			wantType: &v03OpenTelemetryConfig,
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

func TestUnmarshalBatchSpanProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter SpanExporter
	}{
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":5000,"max_export_batch_size":512,"max_queue_size":2048,"schedule_delay":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 5000\nmax_export_batch_size: 512\nmax_queue_size: 2048\nschedule_delay: 1000"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero export_timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 0"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero schedule_delay",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"schedule_delay":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nschedule_delay: 0"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequiredExporter(&BatchSpanProcessor{}),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   newErrUnmarshal(&BatchSpanProcessor{}),
		},
		{
			name:       "invalid export_timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"export_timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name:       "invalid max_export_batch_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_export_batch_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_queue_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid max_queue_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid schedule_delay negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"schedule_delay":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nschedule_delay: -1"),
			wantErrT:   newErrGreaterOrEqualZero("schedule_delay"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := BatchSpanProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = BatchSpanProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
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
			},
		},
		{
			name:    "invalid config",
			input:   "invalid_bool.json",
			wantErr: errors.New(`json: cannot unmarshal string into Go struct field Plain.disabled of type bool`),
		},
		{
			name:    "invalid nil name",
			input:   "invalid_nil_name.json",
			wantErr: errors.New(`json: cannot unmarshal field name in NameStringValuePair required`),
		},
		{
			name:    "invalid nil value",
			input:   "invalid_nil_value.json",
			wantErr: errors.New(`json: cannot unmarshal field value in NameStringValuePair required`),
		},
		{
			name:    "valid v0.2 config",
			input:   "v0.2.json",
			wantErr: errors.New(`json: cannot unmarshal object into Go struct field LogRecordProcessor.logger_provider.processors.batch`),
		},
		{
			name:     "valid v0.3 config",
			input:    "v0.3.json",
			wantType: v03OpenTelemetryConfig,
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

func TestUnmarshalPeriodicMetricReader(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter PushMetricExporter
	}{
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"timeout":5000,"interval":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ntimeout: 5000\ninterval: 1000"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ntimeout: 0"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero interval",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"interval":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ninterval: 0"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequiredExporter(&PeriodicMetricReader{}),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&PeriodicMetricReader{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
		{
			name:       "invalid interval negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"interval":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\ninterval: -1"),
			wantErrT:   newErrGreaterOrEqualZero("interval"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pmr := PeriodicMetricReader{}
			err := pmr.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, pmr.Exporter)

			pmr = PeriodicMetricReader{}
			err = yaml.Unmarshal(tt.yamlConfig, &pmr)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, pmr.Exporter)
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

func TestUnmarshalCardinalityLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErrT   error
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"counter":100,"default":200,"gauge":300,"histogram":400,"observable_counter":500,"observable_gauge":600,"observable_up_down_counter":700,"up_down_counter":800}`),
			yamlConfig: []byte("counter: 100\ndefault: 200\ngauge: 300\nhistogram: 400\nobservable_counter: 500\nobservable_gauge: 600\nobservable_up_down_counter: 700\nup_down_counter: 800"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"default":2000}`),
			yamlConfig: []byte("default: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("counter: !!str 2000"),
			wantErrT:   newErrUnmarshal(&CardinalityLimits{}),
		},
		{
			name:       "invalid counter zero",
			jsonConfig: []byte(`{"counter":0}`),
			yamlConfig: []byte("counter: 0"),
			wantErrT:   newErrGreaterThanZero("counter"),
		},
		{
			name:       "invalid counter negative",
			jsonConfig: []byte(`{"counter":-1}`),
			yamlConfig: []byte("counter: -1"),
			wantErrT:   newErrGreaterThanZero("counter"),
		},
		{
			name:       "invalid default zero",
			jsonConfig: []byte(`{"default":0}`),
			yamlConfig: []byte("default: 0"),
			wantErrT:   newErrGreaterThanZero("default"),
		},
		{
			name:       "invalid default negative",
			jsonConfig: []byte(`{"default":-1}`),
			yamlConfig: []byte("default: -1"),
			wantErrT:   newErrGreaterThanZero("default"),
		},
		{
			name:       "invalid gauge zero",
			jsonConfig: []byte(`{"gauge":0}`),
			yamlConfig: []byte("gauge: 0"),
			wantErrT:   newErrGreaterThanZero("gauge"),
		},
		{
			name:       "invalid gauge negative",
			jsonConfig: []byte(`{"gauge":-1}`),
			yamlConfig: []byte("gauge: -1"),
			wantErrT:   newErrGreaterThanZero("gauge"),
		},
		{
			name:       "invalid histogram zero",
			jsonConfig: []byte(`{"histogram":0}`),
			yamlConfig: []byte("histogram: 0"),
			wantErrT:   newErrGreaterThanZero("histogram"),
		},
		{
			name:       "invalid histogram negative",
			jsonConfig: []byte(`{"histogram":-1}`),
			yamlConfig: []byte("histogram: -1"),
			wantErrT:   newErrGreaterThanZero("histogram"),
		},
		{
			name:       "invalid observable_counter zero",
			jsonConfig: []byte(`{"observable_counter":0}`),
			yamlConfig: []byte("observable_counter: 0"),
			wantErrT:   newErrGreaterThanZero("observable_counter"),
		},
		{
			name:       "invalid observable_counter negative",
			jsonConfig: []byte(`{"observable_counter":-1}`),
			yamlConfig: []byte("observable_counter: -1"),
			wantErrT:   newErrGreaterThanZero("observable_counter"),
		},
		{
			name:       "invalid observable_gauge zero",
			jsonConfig: []byte(`{"observable_gauge":0}`),
			yamlConfig: []byte("observable_gauge: 0"),
			wantErrT:   newErrGreaterThanZero("observable_gauge"),
		},
		{
			name:       "invalid observable_gauge negative",
			jsonConfig: []byte(`{"observable_gauge":-1}`),
			yamlConfig: []byte("observable_gauge: -1"),
			wantErrT:   newErrGreaterThanZero("observable_gauge"),
		},
		{
			name:       "invalid observable_up_down_counter zero",
			jsonConfig: []byte(`{"observable_up_down_counter":0}`),
			yamlConfig: []byte("observable_up_down_counter: 0"),
			wantErrT:   newErrGreaterThanZero("observable_up_down_counter"),
		},
		{
			name:       "invalid observable_up_down_counter negative",
			jsonConfig: []byte(`{"observable_up_down_counter":-1}`),
			yamlConfig: []byte("observable_up_down_counter: -1"),
			wantErrT:   newErrGreaterThanZero("observable_up_down_counter"),
		},
		{
			name:       "invalid up_down_counter zero",
			jsonConfig: []byte(`{"up_down_counter":0}`),
			yamlConfig: []byte("up_down_counter: 0"),
			wantErrT:   newErrGreaterThanZero("up_down_counter"),
		},
		{
			name:       "invalid up_down_counter negative",
			jsonConfig: []byte(`{"up_down_counter":-1}`),
			yamlConfig: []byte("up_down_counter: -1"),
			wantErrT:   newErrGreaterThanZero("up_down_counter"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := CardinalityLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)

			cl = CardinalityLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
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

func TestUnmarshalSpanLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErrT   error
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"attribute_count_limit":100,"attribute_value_length_limit":200,"event_attribute_count_limit":300,"event_count_limit":400,"link_attribute_count_limit":500,"link_count_limit":600}`),
			yamlConfig: []byte("attribute_count_limit: 100\nattribute_value_length_limit: 200\nevent_attribute_count_limit: 300\nevent_count_limit: 400\nlink_attribute_count_limit: 500\nlink_count_limit: 600"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"attribute_value_length_limit":2000}`),
			yamlConfig: []byte("attribute_value_length_limit: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("attribute_count_limit: !!str 2000"),
			wantErrT:   newErrUnmarshal(&SpanLimits{}),
		},
		{
			name:       "invalid attribute_count_limit negative",
			jsonConfig: []byte(`{"attribute_count_limit":-1}`),
			yamlConfig: []byte("attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("attribute_count_limit"),
		},
		{
			name:       "invalid attribute_value_length_limit negative",
			jsonConfig: []byte(`{"attribute_value_length_limit":-1}`),
			yamlConfig: []byte("attribute_value_length_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("attribute_value_length_limit"),
		},
		{
			name:       "invalid event_attribute_count_limit negative",
			jsonConfig: []byte(`{"event_attribute_count_limit":-1}`),
			yamlConfig: []byte("event_attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("event_attribute_count_limit"),
		},
		{
			name:       "invalid event_count_limit negative",
			jsonConfig: []byte(`{"event_count_limit":-1}`),
			yamlConfig: []byte("event_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("event_count_limit"),
		},
		{
			name:       "invalid link_attribute_count_limit negative",
			jsonConfig: []byte(`{"link_attribute_count_limit":-1}`),
			yamlConfig: []byte("link_attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("link_attribute_count_limit"),
		},
		{
			name:       "invalid link_count_limit negative",
			jsonConfig: []byte(`{"link_count_limit":-1}`),
			yamlConfig: []byte("link_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("link_count_limit"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SpanLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)

			cl = SpanLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
		})
	}
}
func ptr[T any](v T) *T {
	return &v
}
