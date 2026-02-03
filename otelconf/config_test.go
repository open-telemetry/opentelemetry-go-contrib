// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.yaml.in/yaml/v3"
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

func TestUnmarshalSimpleLogRecordProcessor(t *testing.T) {
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
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&SimpleLogRecordProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: []"),
			wantErrT:   newErrUnmarshal(&SimpleLogRecordProcessor{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SimpleLogRecordProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = SimpleLogRecordProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalSimpleSpanProcessor(t *testing.T) {
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
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&SimpleSpanProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: []"),
			wantErrT:   newErrUnmarshal(&SimpleSpanProcessor{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SimpleSpanProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = SimpleSpanProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
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
			wantErrT:   newErrRequired(&BatchLogRecordProcessor{}, "exporter"),
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
		wantPropagator     any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "no-configuration",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
			wantPropagator:     propagation.NewCompositeTextMapPropagator(),
		},
		{
			name: "with-configuration",
			cfg: []ConfigurationOption{
				WithContext(t.Context()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					TracerProvider: &TracerProviderJson{},
					MeterProvider:  &MeterProviderJson{},
					LoggerProvider: &LoggerProviderJson{},
					Propagator:     &PropagatorJson{},
				}),
			},
			wantTracerProvider: &sdktrace.TracerProvider{},
			wantMeterProvider:  &sdkmetric.MeterProvider{},
			wantLoggerProvider: &sdklog.LoggerProvider{},
			wantPropagator:     propagation.NewCompositeTextMapPropagator(),
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
			wantPropagator:     propagation.NewCompositeTextMapPropagator(),
		},
	}
	for _, tt := range tests {
		sdk, err := NewSDK(tt.cfg...)
		require.Equal(t, tt.wantErr, err)
		assert.IsType(t, tt.wantTracerProvider, sdk.TracerProvider())
		assert.IsType(t, tt.wantMeterProvider, sdk.MeterProvider())
		assert.IsType(t, tt.wantLoggerProvider, sdk.LoggerProvider())
		assert.IsType(t, tt.wantPropagator, sdk.Propagator())
		require.Equal(t, tt.wantShutdownErr, sdk.Shutdown(t.Context()))
	}
}

func TestNewSDKWithEnvVar(t *testing.T) {
	cfg := []ConfigurationOption{
		WithContext(t.Context()),
		WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
			TracerProvider: nil,
		}),
	}
	// test a non existent file
	t.Setenv(envVarConfigFile, filepath.Join("testdata", "file_missing.yaml"))
	_, err := NewSDK(cfg...)
	require.Error(t, err)
	// test a file that causes a parse error
	t.Setenv(envVarConfigFile, filepath.Join("testdata", "v1.0.0_invalid_nil_name.yaml"))
	_, err = NewSDK(cfg...)
	require.Error(t, err)
	require.ErrorIs(t, err, newErrRequired(&NameStringValuePair{}, "name"))
	// test a valid file, error is returned from the SDK instantiation
	t.Setenv(envVarConfigFile, filepath.Join("testdata", "v1.0.0.yaml"))
	_, err = NewSDK(cfg...)
	require.ErrorIs(t, err, newErrInvalid("otlp_file/development"))
}

var v10OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   ptr(false),
	FileFormat: "1.0-rc.2",
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
							CertificateFile:       ptr("testdata/ca.crt"),
							ClientCertificateFile: ptr("testdata/client.crt"),
							ClientKeyFile:         ptr("testdata/client.key"),
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
							CertificateFile:       ptr("testdata/ca.crt"),
							ClientCertificateFile: ptr("testdata/client.crt"),
							ClientKeyFile:         ptr("testdata/client.key"),
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
							Host:                ptr("localhost"),
							Port:                ptr(9464),
							TranslationStrategy: ptr(ExperimentalPrometheusMetricExporterTranslationStrategyUnderscoreEscapingWithSuffixes),
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
							CertificateFile:             ptr("testdata/ca.crt"),
							ClientCertificateFile:       ptr("testdata/client.crt"),
							ClientKeyFile:               ptr("testdata/client.key"),
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
							CertificateFile:             ptr("testdata/ca.crt"),
							ClientCertificateFile:       ptr("testdata/client.crt"),
							ClientKeyFile:               ptr("testdata/client.key"),
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
			Detectors: []ExperimentalResourceDetector{
				{Container: ExperimentalContainerResourceDetector{}},
				{Host: ExperimentalHostResourceDetector{}},
				{Process: ExperimentalProcessResourceDetector{}},
				{Service: ExperimentalServiceResourceDetector{}},
			},
		},
	},
	TracerProvider: &TracerProviderJson{
		TracerConfiguratorDevelopment: &ExperimentalTracerConfigurator{
			DefaultConfig: &ExperimentalTracerConfig{
				Disabled: ptr(true),
			},
			Tracers: []ExperimentalTracerMatcherAndConfig{
				{
					Config: ptr(ExperimentalTracerConfig{
						Disabled: ptr(false),
					}),
					Name: ptr("io.opentelemetry.contrib.*"),
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
							CertificateFile:       ptr("testdata/ca.crt"),
							ClientCertificateFile: ptr("testdata/client.crt"),
							ClientKeyFile:         ptr("testdata/client.key"),
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
							CertificateFile:       ptr("testdata/ca.crt"),
							ClientCertificateFile: ptr("testdata/client.crt"),
							ClientKeyFile:         ptr("testdata/client.key"),
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

var v100OpenTelemetryConfigEnvParsing = OpenTelemetryConfiguration{
	Disabled:   ptr(false),
	FileFormat: "1.0",
	LogLevel:   ptr("info"),
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       ptr(128),
		AttributeValueLengthLimit: ptr(4096),
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
			{Name: "string_value", Type: &AttributeType{Value: "string"}, Value: "value"},
			{Name: "bool_value", Type: &AttributeType{Value: "bool"}, Value: true},
			{Name: "int_value", Type: &AttributeType{Value: "int"}, Value: 1},
			{Name: "float_value", Type: &AttributeType{Value: "double"}, Value: 1.1},
			{Name: "hex_value", Type: &AttributeType{Value: "int"}, Value: int(48879)},
			{Name: "quoted_string_value", Type: &AttributeType{Value: "string"}, Value: "value"},
			{Name: "quoted_bool_value", Type: &AttributeType{Value: "string"}, Value: "true"},
			{Name: "quoted_int_value", Type: &AttributeType{Value: "string"}, Value: "1"},
			{Name: "quoted_float_value", Type: &AttributeType{Value: "string"}, Value: "1.1"},
			{Name: "quoted_hex_value", Type: &AttributeType{Value: "string"}, Value: "0xbeef"},
			{Name: "alternative_env_syntax", Type: &AttributeType{Value: "string"}, Value: "value"},
			{Name: "invalid_map_value", Type: &AttributeType{Value: "string"}, Value: "value\nkey:value"},
			{Name: "multiple_references_inject", Type: &AttributeType{Value: "string"}, Value: "foo value 1.1"},
			{Name: "undefined_key", Type: &AttributeType{Value: "string"}, Value: nil},
			{Name: "undefined_key_fallback", Type: &AttributeType{Value: "string"}, Value: "fallback"},
			{Name: "env_var_in_key", Type: &AttributeType{Value: "string"}, Value: "value"},
			{Name: "replace_me", Type: &AttributeType{Value: "string"}, Value: "${DO_NOT_REPLACE_ME}"},
			{Name: "undefined_defaults_to_var", Type: &AttributeType{Value: "string"}, Value: "${STRING_VALUE}"},
			{Name: "escaped_does_not_substitute", Type: &AttributeType{Value: "string"}, Value: "${STRING_VALUE}"},
			{Name: "escaped_does_not_substitute_fallback", Type: &AttributeType{Value: "string"}, Value: "${STRING_VALUE:-fallback}"},
			{Name: "escaped_and_substituted_fallback", Type: &AttributeType{Value: "string"}, Value: "${STRING_VALUE:-value}"},
			{Name: "escaped_and_substituted", Type: &AttributeType{Value: "string"}, Value: "$value"},
			{Name: "multiple_escaped_and_not_substituted", Type: &AttributeType{Value: "string"}, Value: "$${STRING_VALUE}"},
			{Name: "undefined_key_with_escape_sequence_in_fallback", Type: &AttributeType{Value: "string"}, Value: "${UNDEFINED_KEY}"},
			{Name: "value_with_escape", Type: &AttributeType{Value: "string"}, Value: "value$$"},
			{Name: "escape_sequence", Type: &AttributeType{Value: "string"}, Value: "a $ b"},
			{Name: "no_escape_sequence", Type: &AttributeType{Value: "string"}, Value: "a $ b"},
		},
		AttributesList: ptr("service.namespace=my-namespace,service.version=1.0.0"),
		// Detectors: &Detectors{
		// 	Attributes: &DetectorsAttributes{
		// 		Excluded: []string{"process.command_args"},
		// 		Included: []string{"process.*"},
		// 	},
		// },
		SchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
	},
}

func TestParseFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType *OpenTelemetryConfiguration
	}{
		{
			name:     "invalid nil name",
			input:    "v1.0.0_invalid_nil_name",
			wantErr:  newErrRequired(&NameStringValuePair{}, "name"),
			wantType: &OpenTelemetryConfiguration{},
		},
		{
			name:     "invalid nil value",
			input:    "v1.0.0_invalid_nil_value",
			wantErr:  newErrRequired(&NameStringValuePair{}, "value"),
			wantType: &OpenTelemetryConfiguration{},
		},
		{
			name:     "valid v0.2 config",
			input:    "v0.2",
			wantErr:  newErrUnmarshal(&OpenTelemetryConfiguration{}),
			wantType: &OpenTelemetryConfiguration{},
		},
		{
			name:     "valid v0.3 config",
			input:    "v0.3",
			wantErr:  newErrUnmarshal(&TextMapPropagator{}),
			wantType: &OpenTelemetryConfiguration{},
		},
		{
			name:     "valid v1.0.0 config",
			input:    "v1.0.0",
			wantType: &v10OpenTelemetryConfig,
		},
	}

	for _, tt := range tests {
		t.Run("yaml:"+tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", fmt.Sprintf("%s.yaml", tt.input)))
			require.NoError(t, err)

			got, err := ParseYAML(b)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.Equal(t, tt.wantType, got)
			}
		})
		t.Run("json: "+tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", fmt.Sprintf("%s.json", tt.input)))
			require.NoError(t, err)

			var got OpenTelemetryConfiguration
			err = json.Unmarshal(b, &got)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantType, &got)
		})
	}
}

func TestUnmarshalOpenTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		jsonConfig []byte
		yamlConfig []byte
		wantErr    error
		wantType   OpenTelemetryConfiguration
	}{
		{
			name:       "valid defaults config",
			jsonConfig: []byte(`{"file_format": "1.0"}`),
			yamlConfig: []byte("file_format: 1.0"),
			wantType: OpenTelemetryConfiguration{
				Disabled:   ptr(false),
				FileFormat: "1.0",
				LogLevel:   ptr("info"),
			},
		},
		{
			name:       "invalid config missing required file_format",
			jsonConfig: []byte(`{"disabled": false}`),
			yamlConfig: []byte("disabled: false"),
			wantErr:    newErrRequired(&OpenTelemetryConfiguration{}, "file_format"),
		},
		{
			name:       "file_format invalid",
			jsonConfig: []byte(`{"file_format":[], "disabled": false}`),
			yamlConfig: []byte("file_format: []\ndisabled: false"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "invalid config",
			jsonConfig: []byte(`{"file_format": "yaml", "disabled": "notabool"}`),
			yamlConfig: []byte("file_format: []\ndisabled: notabool"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("disabled: []\nconsole: {}\nfile_format: str"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "resource invalid",
			jsonConfig: []byte(`{"resource":[], "file_format": "1.0"}`),
			yamlConfig: []byte("resource: []\nfile_format: 1.0"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "attribute_limits invalid",
			jsonConfig: []byte(`{"attribute_limits":[], "file_format": "1.0"}`),
			yamlConfig: []byte("attribute_limits: []\nfile_format: 1.0"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "instrumentation invalid",
			jsonConfig: []byte(`{"instrumentation/development":[], "file_format": "1.0"}`),
			yamlConfig: []byte("instrumentation/development: []\nfile_format: 1.0"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
		{
			name:       "log_level invalid",
			jsonConfig: []byte(`{"log_level":[], "file_format": "1.0"}`),
			yamlConfig: []byte("log_level: []\nfile_format: 1.0"),
			wantErr:    newErrUnmarshal(&OpenTelemetryConfiguration{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OpenTelemetryConfiguration{}
			err := got.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantType, got)

			got = OpenTelemetryConfiguration{}
			err = yaml.Unmarshal(tt.yamlConfig, &got)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantType, got)
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
			wantErrT:   newErrRequired(&BatchSpanProcessor{}, "exporter"),
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

func TestParseYAMLWithEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType any
	}{
		{
			name:     "valid v1.0.0 config with env vars",
			input:    "v1.0.0_env_var.yaml",
			wantType: &v100OpenTelemetryConfigEnvParsing,
		},
	}

	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT", "4096")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("STRING_VALUE", "value")
	t.Setenv("BOOL_VALUE", "true")
	t.Setenv("INT_VALUE", "1")
	t.Setenv("FLOAT_VALUE", "1.1")
	t.Setenv("HEX_VALUE", "0xbeef")                       // A valid integer value (i.e. 3735928559) written in hexadecimal
	t.Setenv("INVALID_MAP_VALUE", "value\\nkey:value")    // An invalid attempt to inject a map key into the YAML
	t.Setenv("ENV_VAR_IN_KEY", "env_var_in_key")          // An env var in key
	t.Setenv("DO_NOT_REPLACE_ME", "Never use this value") // An unused environment variable
	t.Setenv("REPLACE_ME", "${DO_NOT_REPLACE_ME}")        // A valid replacement text, used verbatim, not replaced with "Never use this value"
	t.Setenv("VALUE_WITH_ESCAPE", "value$$")              // A valid replacement text, used verbatim, not replaced with "Never use this value"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", tt.input))
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
			wantErrT:   newErrRequired(&PeriodicMetricReader{}, "exporter"),
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
		wantErr     error
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
			wantErr:     newErrInvalid("invalid headers_list"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headersMap, err := createHeadersConfig(tt.headers, tt.headersList)
			require.ErrorIs(t, err, tt.wantErr)
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

func TestUnmarshalOTLPHttpExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPHttpExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPHttpExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPHttpExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPHttpExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPHttpExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPHttpExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPHttpExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPGrpcExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPGrpcExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPGrpcExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPGrpcExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPGrpcExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPGrpcExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPGrpcExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPGrpcExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPHttpMetricExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPHttpMetricExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPHttpMetricExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPHttpMetricExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPHttpMetricExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPHttpMetricExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPHttpMetricExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPHttpMetricExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPGrpcMetricExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPGrpcMetricExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPGrpcMetricExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPGrpcMetricExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPGrpcMetricExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPGrpcMetricExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPGrpcMetricExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPGrpcMetricExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalZipkinSpanExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter ZipkinSpanExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:9000"}`),
			yamlConfig:   []byte("endpoint: localhost:9000\n"),
			wantExporter: ZipkinSpanExporter{Endpoint: ptr("localhost:9000")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&ZipkinSpanExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:9000", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:9000\ntimeout: 0"),
			wantExporter: ZipkinSpanExporter{Endpoint: ptr("localhost:9000"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:9000\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&ZipkinSpanExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:9000", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:9000\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := ZipkinSpanExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = ZipkinSpanExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalAttributeNameValueType(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		yamlConfig             []byte
		jsonConfig             []byte
		wantErrT               error
		wantAttributeNameValue AttributeNameValue
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&AttributeNameValue{}),
		},
		{
			name:       "missing required name field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&AttributeNameValue{}, "name"),
		},
		{
			name:       "missing required value field",
			jsonConfig: []byte(`{"name":"test"}`),
			yamlConfig: []byte("name: test"),
			wantErrT:   newErrRequired(&AttributeNameValue{}, "value"),
		},
		{
			name:       "valid string value",
			jsonConfig: []byte(`{"name":"test", "value": "test-val", "type": "string"}`),
			yamlConfig: []byte("name: test\nvalue: test-val\ntype: string\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: "test-val",
				Type:  &AttributeType{Value: "string"},
			},
		},
		{
			name:       "valid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{"test-val", "test-val-2"},
				Type:  &AttributeType{Value: "string_array"},
			},
		},
		{
			name:       "valid bool value",
			jsonConfig: []byte(`{"name":"test", "value": true, "type": "bool"}`),
			yamlConfig: []byte("name: test\nvalue: true\ntype: bool\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: true,
				Type:  &AttributeType{Value: "bool"},
			},
		},
		{
			name:       "valid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{"test-val", "test-val-2"},
				Type:  &AttributeType{Value: "string_array"},
			},
		},
		{
			name:       "valid int value",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "int"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: int\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: int(1),
				Type:  &AttributeType{Value: "int"},
			},
		},
		{
			name:       "valid int_array value",
			jsonConfig: []byte(`{"name":"test", "value": [1, 2], "type": "int_array"}`),
			yamlConfig: []byte("name: test\nvalue: [1, 2]\ntype: int_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{1, 2},
				Type:  &AttributeType{Value: "int_array"},
			},
		},
		{
			name:       "valid double value",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "double"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: double\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: float64(1),
				Type:  &AttributeType{Value: "double"},
			},
		},
		{
			name:       "valid double_array value",
			jsonConfig: []byte(`{"name":"test", "value": [1, 2], "type": "double_array"}`),
			yamlConfig: []byte("name: test\nvalue: [1.0, 2.0]\ntype: double_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{float64(1), float64(2)},
				Type:  &AttributeType{Value: "double_array"},
			},
		},
		{
			name:       "invalid type",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "float"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: float\n"),
			wantErrT:   newErrInvalid("unexpected value type"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := AttributeNameValue{}
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantAttributeNameValue, val)

			val = AttributeNameValue{}
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantAttributeNameValue, val)
		})
	}
}

func TestUnmarshalNameStringValuePairType(t *testing.T) {
	for _, tt := range []struct {
		name                    string
		yamlConfig              []byte
		jsonConfig              []byte
		wantErrT                error
		wantNameStringValuePair NameStringValuePair
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&NameStringValuePair{}),
		},
		{
			name:       "missing required name field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&NameStringValuePair{}, "name"),
		},
		{
			name:       "missing required value field",
			jsonConfig: []byte(`{"name":"test"}`),
			yamlConfig: []byte("name: test"),
			wantErrT:   newErrRequired(&NameStringValuePair{}, "value"),
		},
		{
			name:       "invalid array name",
			jsonConfig: []byte(`{"name":[], "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: []\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantErrT:   newErrUnmarshal(&NameStringValuePair{}),
		},
		{
			name:       "valid string value",
			jsonConfig: []byte(`{"name":"test", "value": "test-val", "type": "string"}`),
			yamlConfig: []byte("name: test\nvalue: test-val\ntype: string\n"),
			wantNameStringValuePair: NameStringValuePair{
				Name:  "test",
				Value: ptr("test-val"),
			},
		},
		{
			name:       "invalid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantErrT:   newErrUnmarshal(&NameStringValuePair{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := NameStringValuePair{}
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantNameStringValuePair, val)

			val = NameStringValuePair{}
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantNameStringValuePair, val)
		})
	}
}

func TestUnmarshalInstrumentType(t *testing.T) {
	var instrumentType InstrumentType
	for _, tt := range []struct {
		name               string
		yamlConfig         []byte
		jsonConfig         []byte
		wantErrT           error
		wantInstrumentType InstrumentType
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&instrumentType),
		},
		{
			name:       "invalid instrument type",
			jsonConfig: []byte(`"test"`),
			yamlConfig: []byte("test"),
			wantErrT:   newErrInvalid(`invalid selector (expected one of []interface {}{"counter", "gauge", "histogram", "observable_counter", "observable_gauge", "observable_up_down_counter", "up_down_counter"}): "test""`),
		},
		{
			name:               "valid instrument type",
			jsonConfig:         []byte(`"counter"`),
			yamlConfig:         []byte("counter"),
			wantInstrumentType: InstrumentTypeCounter,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := InstrumentType("")
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantInstrumentType, val)

			val = InstrumentType("")
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantInstrumentType, val)
		})
	}
}

func TestUnmarshalExperimentalPeerInstrumentationServiceMappingElemType(t *testing.T) {
	for _, tt := range []struct {
		name                                                  string
		yamlConfig                                            []byte
		jsonConfig                                            []byte
		wantErrT                                              error
		wantExperimentalPeerInstrumentationServiceMappingElem ExperimentalPeerInstrumentationServiceMappingElem
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("peer: []\nservice: true"),
			wantErrT:   newErrUnmarshal(&ExperimentalPeerInstrumentationServiceMappingElem{}),
		},
		{
			name:       "missing required peer field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&ExperimentalPeerInstrumentationServiceMappingElem{}, "peer"),
		},
		{
			name:       "missing required service field",
			jsonConfig: []byte(`{"peer":"test"}`),
			yamlConfig: []byte("peer: test"),
			wantErrT:   newErrRequired(&ExperimentalPeerInstrumentationServiceMappingElem{}, "service"),
		},
		{
			name:       "invalid string_array peer",
			jsonConfig: []byte(`{"peer":[], "service": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("peer: []\nservice: [test-val, test-val-2]\ntype: string_array\n"),
			wantErrT:   newErrUnmarshal(&ExperimentalPeerInstrumentationServiceMappingElem{}),
		},
		{
			name:       "valid string service",
			jsonConfig: []byte(`{"peer":"test", "service": "test-val"}`),
			yamlConfig: []byte("peer: test\nservice: test-val"),
			wantExperimentalPeerInstrumentationServiceMappingElem: ExperimentalPeerInstrumentationServiceMappingElem{
				Peer:    "test",
				Service: "test-val",
			},
		},
		{
			name:       "invalid string_array service",
			jsonConfig: []byte(`{"peer":"test", "service": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("peer: test\nservice: [test-val, test-val-2]\ntype: string_array\n"),
			wantErrT:   newErrUnmarshal(&ExperimentalPeerInstrumentationServiceMappingElem{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := ExperimentalPeerInstrumentationServiceMappingElem{}
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExperimentalPeerInstrumentationServiceMappingElem, val)

			val = ExperimentalPeerInstrumentationServiceMappingElem{}
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExperimentalPeerInstrumentationServiceMappingElem, val)
		})
	}
}

func TestUnmarshalExporterDefaultHistogramAggregation(t *testing.T) {
	var exporterDefaultHistogramAggregation ExporterDefaultHistogramAggregation
	for _, tt := range []struct {
		name                                    string
		yamlConfig                              []byte
		jsonConfig                              []byte
		wantErrT                                error
		wantExporterDefaultHistogramAggregation ExporterDefaultHistogramAggregation
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&exporterDefaultHistogramAggregation),
		},
		{
			name:       "invalid histogram aggregation",
			jsonConfig: []byte(`"test"`),
			yamlConfig: []byte("test"),
			wantErrT:   newErrInvalid(`invalid histogram aggregation (expected one of []interface {}{"explicit_bucket_histogram", "base2_exponential_bucket_histogram"}): "test""`),
		},
		{
			name:                                    "valid histogram aggregation",
			jsonConfig:                              []byte(`"base2_exponential_bucket_histogram"`),
			yamlConfig:                              []byte("base2_exponential_bucket_histogram"),
			wantExporterDefaultHistogramAggregation: ExporterDefaultHistogramAggregationBase2ExponentialBucketHistogram,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := ExporterDefaultHistogramAggregation("")
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporterDefaultHistogramAggregation, val)

			val = ExporterDefaultHistogramAggregation("")
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporterDefaultHistogramAggregation, val)
		})
	}
}

func TestUnmarshalPullMetricReader(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter PullMetricExporter
	}{
		{
			name:         "valid with proemtheus exporter",
			jsonConfig:   []byte(`{"exporter":{"prometheus/development":{}}}`),
			yamlConfig:   []byte("exporter:\n  prometheus/development: {}"),
			wantExporter: PullMetricExporter{PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&PullMetricReader{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  prometheus/development: []"),
			wantErrT:   newErrUnmarshal(&PullMetricReader{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := PullMetricReader{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = PullMetricReader{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalResourceJson(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantResource ResourceJson
	}{
		{
			name:       "valid with all detectors",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"container": null},{"host": null},{"process": null},{"service": null}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - container:\n    - host:\n    - process:\n    - service:"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{
							Container: ExperimentalContainerResourceDetector{},
						},
						{
							Host: ExperimentalHostResourceDetector{},
						},
						{
							Process: ExperimentalProcessResourceDetector{},
						},
						{
							Service: ExperimentalServiceResourceDetector{},
						},
					},
				},
			},
		},
		{
			name:       "valid non-nil with all detectors",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"container": {}},{"host": {}},{"process": {}},{"service": {}}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - container: {}\n    - host: {}\n    - process: {}\n    - service: {}"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{
							Container: ExperimentalContainerResourceDetector{},
						},
						{
							Host: ExperimentalHostResourceDetector{},
						},
						{
							Process: ExperimentalProcessResourceDetector{},
						},
						{
							Service: ExperimentalServiceResourceDetector{},
						},
					},
				},
			},
		},
		{
			name:       "invalid container detector",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"container": 1}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - container: 1"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{},
					},
				},
			},
			wantErrT: newErrUnmarshal(&ExperimentalResourceDetector{}),
		},
		{
			name:       "invalid host detector",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"host": 1}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - host: 1"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{},
					},
				},
			},
			wantErrT: newErrUnmarshal(&ExperimentalResourceDetector{}),
		},
		{
			name:       "invalid service detector",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"service": 1}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - service: 1"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{},
					},
				},
			},
			wantErrT: newErrUnmarshal(&ExperimentalResourceDetector{}),
		},
		{
			name:       "invalid process detector",
			jsonConfig: []byte(`{"detection/development": {"detectors": [{"process": 1}]}}`),
			yamlConfig: []byte("detection/development:\n  detectors:\n    - process: 1"),
			wantResource: ResourceJson{
				DetectionDevelopment: &ExperimentalResourceDetection{
					Detectors: []ExperimentalResourceDetector{
						{},
					},
				},
			},
			wantErrT: newErrUnmarshal(&ExperimentalResourceDetector{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceJson{}
			err := json.Unmarshal(tt.jsonConfig, &r)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantResource, r)

			r = ResourceJson{}
			err = yaml.Unmarshal(tt.yamlConfig, &r)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantResource, r)
		})
	}
}
