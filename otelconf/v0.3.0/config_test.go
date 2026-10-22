// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.yaml.in/yaml/v2"
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

var v03OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   new(false),
	FileFormat: new("0.3"),
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       new(128),
		AttributeValueLengthLimit: new(4096),
	},
	Instrumentation: &Instrumentation{
		Cpp: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Dotnet: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Erlang: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		General: &GeneralInstrumentation{
			Http: &GeneralInstrumentationHttp{
				Client: &GeneralInstrumentationHttpClient{
					RequestCapturedHeaders:  []string{"Content-Type", "Accept"},
					ResponseCapturedHeaders: []string{"Content-Type", "Content-Encoding"},
				},
				Server: &GeneralInstrumentationHttpServer{
					RequestCapturedHeaders:  []string{"Content-Type", "Accept"},
					ResponseCapturedHeaders: []string{"Content-Type", "Content-Encoding"},
				},
			},
			Peer: &GeneralInstrumentationPeer{
				ServiceMapping: []GeneralInstrumentationPeerServiceMappingElem{
					{Peer: "1.2.3.4", Service: "FooService"},
					{Peer: "2.3.4.5", Service: "BarService"},
				},
			},
		},
		Go: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Java: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Js: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Php: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Python: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Ruby: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Rust: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
		Swift: LanguageSpecificInstrumentation{
			"example": map[string]any{
				"property": "value",
			},
		},
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
							Endpoint:          new("http://localhost:4318/v1/logs"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: new("1234")},
							},
							HeadersList: new("api-key=1234"),
							Insecure:    new(false),
							Protocol:    new("http/protobuf"),
							Timeout:     new(10000),
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
				Producers: []MetricProducer{
					{Opencensus: MetricProducerOpencensus{}},
				},
				Pull: &PullMetricReader{
					Exporter: PullMetricExporter{
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
				Producers: []MetricProducer{
					{},
				},
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Certificate:                 new("/app/cert.pem"),
							ClientCertificate:           new("/app/cert.pem"),
							ClientKey:                   new("/app/cert.pem"),
							Compression:                 new("gzip"),
							DefaultHistogramAggregation: new(OTLPMetricDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    new("http://localhost:4318/v1/metrics"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: new("1234")},
							},
							HeadersList:           new("api-key=1234"),
							Insecure:              new(false),
							Protocol:              new("http/protobuf"),
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
					Exporter: PushMetricExporter{
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
					AttributeKeys: &IncludeExclude{
						Included: []string{"key1", "key2"},
						Excluded: []string{"key3"},
					},
					Description: new("new_description"),
					Name:        new("new_instrument_name"),
				},
			},
		},
	},
	Propagator: &Propagator{
		Composite: []*string{new("tracecontext"), new("baggage"), new("b3"), new("b3multi"), new("jaeger"), new("xray"), new("ottrace")},
	},
	Resource: &Resource{
		Attributes: []AttributeNameValue{
			{Name: "service.name", Value: "unknown_service"},
			{Name: "string_key", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "bool_key", Type: &AttributeNameValueType{Value: "bool"}, Value: true},
			{Name: "int_key", Type: &AttributeNameValueType{Value: "int"}, Value: 1},
			{Name: "double_key", Type: &AttributeNameValueType{Value: "double"}, Value: 1.1},
			{Name: "string_array_key", Type: &AttributeNameValueType{Value: "string_array"}, Value: []any{"value1", "value2"}},
			{Name: "bool_array_key", Type: &AttributeNameValueType{Value: "bool_array"}, Value: []any{true, false}},
			{Name: "int_array_key", Type: &AttributeNameValueType{Value: "int_array"}, Value: []any{1, 2}},
			{Name: "double_array_key", Type: &AttributeNameValueType{Value: "double_array"}, Value: []any{1.1, 2.2}},
		},
		AttributesList: new("service.namespace=my-namespace,service.version=1.0.0"),
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
							Endpoint:          new("http://localhost:4318/v1/traces"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: new("1234")},
							},
							HeadersList: new("api-key=1234"),
							Insecure:    new(false),
							Protocol:    new("http/protobuf"),
							Timeout:     new(10000),
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
							Endpoint: new("http://localhost:9411/api/v2/spans"),
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

var v03OpenTelemetryConfigEnvParsing = OpenTelemetryConfiguration{
	Disabled:   new(false),
	FileFormat: new("0.3"),
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       new(128),
		AttributeValueLengthLimit: new(4096),
	},
	Resource: &Resource{
		Attributes: []AttributeNameValue{
			{Name: "service.name", Value: "unknown_service"},
			{Name: "string_key", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "bool_key", Type: &AttributeNameValueType{Value: "bool"}, Value: true},
			{Name: "int_key", Type: &AttributeNameValueType{Value: "int"}, Value: 1},
			{Name: "double_key", Type: &AttributeNameValueType{Value: "double"}, Value: 1.1},
			{Name: "string_array_key", Type: &AttributeNameValueType{Value: "string_array"}, Value: []any{"value1", "value2"}},
			{Name: "bool_array_key", Type: &AttributeNameValueType{Value: "bool_array"}, Value: []any{true, false}},
			{Name: "int_array_key", Type: &AttributeNameValueType{Value: "int_array"}, Value: []any{1, 2}},
			{Name: "double_array_key", Type: &AttributeNameValueType{Value: "double_array"}, Value: []any{1.1, 2.2}},
			{Name: "string_value", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "bool_value", Type: &AttributeNameValueType{Value: "bool"}, Value: true},
			{Name: "int_value", Type: &AttributeNameValueType{Value: "int"}, Value: 1},
			{Name: "float_value", Type: &AttributeNameValueType{Value: "double"}, Value: 1.1},
			{Name: "hex_value", Type: &AttributeNameValueType{Value: "int"}, Value: int(48879)},
			{Name: "quoted_string_value", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "quoted_bool_value", Type: &AttributeNameValueType{Value: "string"}, Value: "true"},
			{Name: "quoted_int_value", Type: &AttributeNameValueType{Value: "string"}, Value: "1"},
			{Name: "quoted_float_value", Type: &AttributeNameValueType{Value: "string"}, Value: "1.1"},
			{Name: "quoted_hex_value", Type: &AttributeNameValueType{Value: "string"}, Value: "0xbeef"},
			{Name: "alternative_env_syntax", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "invalid_map_value", Type: &AttributeNameValueType{Value: "string"}, Value: "value\nkey:value"},
			{Name: "multiple_references_inject", Type: &AttributeNameValueType{Value: "string"}, Value: "foo value 1.1"},
			{Name: "undefined_key", Type: &AttributeNameValueType{Value: "string"}, Value: nil},
			{Name: "undefined_key_fallback", Type: &AttributeNameValueType{Value: "string"}, Value: "fallback"},
			{Name: "env_var_in_key", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "replace_me", Type: &AttributeNameValueType{Value: "string"}, Value: "${DO_NOT_REPLACE_ME}"},
			{Name: "undefined_defaults_to_var", Type: &AttributeNameValueType{Value: "string"}, Value: "${STRING_VALUE}"},
			{Name: "escaped_does_not_substitute", Type: &AttributeNameValueType{Value: "string"}, Value: "${STRING_VALUE}"},
			{Name: "escaped_does_not_substitute_fallback", Type: &AttributeNameValueType{Value: "string"}, Value: "${STRING_VALUE:-fallback}"},
			{Name: "escaped_and_substituted_fallback", Type: &AttributeNameValueType{Value: "string"}, Value: "${STRING_VALUE:-value}"},
			{Name: "escaped_and_substituted", Type: &AttributeNameValueType{Value: "string"}, Value: "$value"},
			{Name: "multiple_escaped_and_not_substituted", Type: &AttributeNameValueType{Value: "string"}, Value: "$${STRING_VALUE}"},
			{Name: "undefined_key_with_escape_sequence_in_fallback", Type: &AttributeNameValueType{Value: "string"}, Value: "${UNDEFINED_KEY}"},
			{Name: "value_with_escape", Type: &AttributeNameValueType{Value: "string"}, Value: "value$$"},
			{Name: "escape_sequence", Type: &AttributeNameValueType{Value: "string"}, Value: "a $ b"},
			{Name: "no_escape_sequence", Type: &AttributeNameValueType{Value: "string"}, Value: "a $ b"},
		},
		AttributesList: new("service.namespace=my-namespace,service.version=1.0.0"),
		Detectors: &Detectors{
			Attributes: &DetectorsAttributes{
				Excluded: []string{"process.command_args"},
				Included: []string{"process.*"},
			},
		},
		SchemaUrl: new("https://opentelemetry.io/schemas/1.16.0"),
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
				FileFormat: new("0.1"),
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

func TestParseYAMLWithEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType any
	}{
		{
			name:     "valid v0.3 config with env vars",
			input:    "v0.3-env-var.yaml",
			wantType: &v03OpenTelemetryConfigEnvParsing,
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
		wantErr  []error
		wantType any
	}{
		{
			name:    "valid JSON config",
			input:   `valid_empty.json`,
			wantErr: nil,
			wantType: OpenTelemetryConfiguration{
				Disabled:   new(false),
				FileFormat: new("0.1"),
			},
		},
		{
			name:    "invalid config",
			input:   "invalid_bool.json",
			wantErr: []error{errors.New(`json: cannot unmarshal string into Go struct field Plain.disabled of type bool`)},
		},
		{
			name:    "invalid nil name",
			input:   "invalid_nil_name.json",
			wantErr: []error{errors.New(`json: cannot unmarshal field name in NameStringValuePair required`)},
		},
		{
			name:    "invalid nil value",
			input:   "invalid_nil_value.json",
			wantErr: []error{errors.New(`json: cannot unmarshal field value in NameStringValuePair required`)},
		},
		{
			name:  "valid v0.2 config",
			input: "v0.2.json",
			wantErr: []error{
				errors.New(`json: cannot unmarshal object into Go struct field LogRecordProcessor.logger_provider.processors.batch`),
				errors.New(`json: cannot unmarshal object into Go struct field Plain.headers of type []otelconf.NameStringValuePair`),
			},
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
				require.Condition(t, func() bool {
					for _, wantErr := range tt.wantErr {
						if strings.Contains(err.Error(), wantErr.Error()) {
							return true
						}
					}
					return false
				}, "error %q does not contain any expected message: %v", err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantType, got)
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
			headersList: new("a=b,c=d"),
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
					Value: new("b"),
				},
				{
					Name:  "c",
					Value: new("d"),
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
					Value: new("b"),
				},
			},
			headersList: new("c=d"),
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
					Value: new("b"),
				},
				{
					Name:  "c",
					Value: new("override"),
				},
			},
			headersList: new("c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "override",
			},
		},
		{
			name:        "invalid headerslist",
			headersList: new("==="),
			wantErr:     "invalid headers list: invalid key: \"\"",
		},
		{
			name: "empty header name",
			headers: []NameStringValuePair{
				{
					Name:  "",
					Value: new("token"),
				},
			},
			wantErr: "invalid header: empty name",
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

func TestUnmarshalCustomSpanExporter(t *testing.T) {
	jsonConfig := []byte(`{
		"file_format": "1.0",
		"tracer_provider": {
			"processors": [{
				"simple": {
					"exporter": {
						"custom": {
							"endpoint": "http://localhost:4318"
						}
					}
				}
			}]
		}
	}`)

	yamlConfig := []byte(`
file_format: "1.0"
tracer_provider:
  processors:
    - simple:
        exporter:
          custom:
            endpoint: http://localhost:4318
`)

	t.Run("JSON", func(t *testing.T) {
		var cfg OpenTelemetryConfiguration
		require.NoError(t, json.Unmarshal(jsonConfig, &cfg))

		exporter := cfg.TracerProvider.Processors[0].Simple.Exporter
		properties, ok := exporter.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "http://localhost:4318", custom["endpoint"])
	})

	t.Run("YAML", func(t *testing.T) {
		cfg, err := ParseYAML(yamlConfig)
		require.NoError(t, err)

		exporter := cfg.TracerProvider.Processors[0].Simple.Exporter
		properties, ok := exporter.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "http://localhost:4318", custom["endpoint"])
	})
}

func TestUnmarshalCustomLogRecordExporter(t *testing.T) {
	jsonConfig := []byte(`{
		"file_format": "1.0",
		"logger_provider": {
			"processors": [{
				"simple": {
					"exporter": {
						"custom": {
							"endpoint": "http://localhost:4318"
						}
					}
				}
			}]
		}
	}`)

	yamlConfig := []byte(`
file_format: "1.0"
logger_provider:
  processors:
    - simple:
        exporter:
          custom:
            endpoint: http://localhost:4318
`)

	t.Run("JSON", func(t *testing.T) {
		var cfg OpenTelemetryConfiguration
		require.NoError(t, json.Unmarshal(jsonConfig, &cfg))

		exporter := cfg.LoggerProvider.Processors[0].Simple.Exporter
		properties, ok := exporter.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "http://localhost:4318", custom["endpoint"])
	})

	t.Run("YAML", func(t *testing.T) {
		cfg, err := ParseYAML(yamlConfig)
		require.NoError(t, err)

		exporter := cfg.LoggerProvider.Processors[0].Simple.Exporter
		properties, ok := exporter.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "http://localhost:4318", custom["endpoint"])
	})
}

func TestUnmarshalCustomPropagator(t *testing.T) {
	jsonConfig := []byte(`{
		"file_format": "1.0",
		"propagator": {
			"custom": {
				"format": "custom-format"
			}
		}
	}`)

	yamlConfig := []byte(`
file_format: "1.0"
propagator:
  custom:
    format: custom-format
`)

	t.Run("JSON", func(t *testing.T) {
		var cfg OpenTelemetryConfiguration
		require.NoError(t, json.Unmarshal(jsonConfig, &cfg))

		propagator := cfg.Propagator
		require.NotNil(t, propagator)
		properties, ok := propagator.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "custom-format", custom["format"])
	})

	t.Run("YAML", func(t *testing.T) {
		cfg, err := ParseYAML(yamlConfig)
		require.NoError(t, err)

		propagator := cfg.Propagator
		require.NotNil(t, propagator)
		properties, ok := propagator.AdditionalProperties.(map[string]any)
		require.True(t, ok, "AdditionalProperties should be a map")
		require.Contains(t, properties, "custom")

		custom, ok := properties["custom"].(map[string]any)
		require.True(t, ok, "custom configuration should be a map")
		assert.Equal(t, "custom-format", custom["format"])
	})
}

func TestAdditionalPropertiesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		new  func() (any, func() any)
	}{
		{
			name: "SpanExporter",
			new: func() (any, func() any) {
				v := &SpanExporter{
					AdditionalProperties: map[string]any{
						"custom": map[string]any{"endpoint": "localhost"},
					},
				}
				return v, func() any { return &SpanExporter{} }
			},
		},
		{
			name: "LogRecordExporter",
			new: func() (any, func() any) {
				v := &LogRecordExporter{
					AdditionalProperties: map[string]any{
						"custom": map[string]any{"endpoint": "localhost"},
					},
				}
				return v, func() any { return &LogRecordExporter{} }
			},
		},
		{
			name: "Propagator",
			new: func() (any, func() any) {
				v := &Propagator{
					AdditionalProperties: map[string]any{
						"custom": map[string]any{"format": "custom-format"},
					},
				}
				return v, func() any { return &Propagator{} }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, newTarget := tt.new()

			t.Run("JSON", func(t *testing.T) {
				data, err := json.Marshal(original)
				require.NoError(t, err)

				target := newTarget()
				require.NoError(t, json.Unmarshal(data, target))

				roundTrip, err := json.Marshal(target)
				require.NoError(t, err)

				assert.JSONEq(t, string(data), string(roundTrip))
			})

			t.Run("YAML", func(t *testing.T) {
				data, err := yaml.Marshal(original)
				require.NoError(t, err)

				target := newTarget()
				require.NoError(t, yaml.Unmarshal(data, target))

				roundTrip, err := yaml.Marshal(target)
				require.NoError(t, err)

				var want any
				require.NoError(t, yaml.Unmarshal(data, &want))

				var got any
				require.NoError(t, yaml.Unmarshal(roundTrip, &got))

				assert.Equal(t, want, got)
			})
		})
	}
}

func TestZeroValueAdditionalPropertiesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		new  func() (any, func() any)
	}{
		{
			name: "SpanExporter",
			new: func() (any, func() any) {
				return &SpanExporter{}, func() any { return &SpanExporter{} }
			},
		},
		{
			name: "LogRecordExporter",
			new: func() (any, func() any) {
				return &LogRecordExporter{}, func() any { return &LogRecordExporter{} }
			},
		},
		{
			name: "Propagator",
			new: func() (any, func() any) {
				return &Propagator{}, func() any { return &Propagator{} }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, newTarget := tt.new()

			data, err := json.Marshal(original)
			require.NoError(t, err)

			target := newTarget()
			require.NoError(t, json.Unmarshal(data, target))

			switch got := target.(type) {
			case *SpanExporter:
				assert.Nil(t, got.AdditionalProperties)
			case *LogRecordExporter:
				assert.Nil(t, got.AdditionalProperties)
			case *Propagator:
				assert.Nil(t, got.AdditionalProperties)
			}
		})
	}
}

func TestUnmarshalEmptyInputPreservesReceiver(t *testing.T) {
	t.Run("SpanExporter", func(t *testing.T) {
		endpoint := "http://localhost:4318"
		protocol := "http/protobuf"

		original := &SpanExporter{
			OTLP: &OTLP{
				Endpoint: &endpoint,
				Protocol: &protocol,
			},
		}

		require.NoError(t, json.Unmarshal([]byte(`{}`), original))
		require.NotNil(t, original.OTLP)
		require.NotNil(t, original.OTLP.Endpoint)
		require.NotNil(t, original.OTLP.Protocol)
		assert.Equal(t, endpoint, *original.OTLP.Endpoint)
		assert.Equal(t, protocol, *original.OTLP.Protocol)

		require.NoError(t, json.Unmarshal([]byte(`null`), original))
		require.NotNil(t, original.OTLP)
		require.NotNil(t, original.OTLP.Endpoint)
		require.NotNil(t, original.OTLP.Protocol)
		assert.Equal(t, endpoint, *original.OTLP.Endpoint)
		assert.Equal(t, protocol, *original.OTLP.Protocol)
	})

	t.Run("LogRecordExporter", func(t *testing.T) {
		endpoint := "http://localhost:4318"
		protocol := "http/protobuf"

		original := &LogRecordExporter{
			OTLP: &OTLP{
				Endpoint: &endpoint,
				Protocol: &protocol,
			},
		}

		require.NoError(t, json.Unmarshal([]byte(`{}`), original))
		require.NotNil(t, original.OTLP)
		require.NotNil(t, original.OTLP.Endpoint)
		require.NotNil(t, original.OTLP.Protocol)
		assert.Equal(t, endpoint, *original.OTLP.Endpoint)
		assert.Equal(t, protocol, *original.OTLP.Protocol)

		require.NoError(t, json.Unmarshal([]byte(`null`), original))
		require.NotNil(t, original.OTLP)
		require.NotNil(t, original.OTLP.Endpoint)
		require.NotNil(t, original.OTLP.Protocol)
		assert.Equal(t, endpoint, *original.OTLP.Endpoint)
		assert.Equal(t, protocol, *original.OTLP.Protocol)
	})

	t.Run("Propagator", func(t *testing.T) {
		composite := "tracecontext"
		original := &Propagator{
			Composite: []*string{&composite},
		}

		require.NoError(t, json.Unmarshal([]byte(`{}`), original))
		require.Equal(t, []*string{&composite}, original.Composite)

		require.NoError(t, json.Unmarshal([]byte(`null`), original))
		require.Equal(t, []*string{&composite}, original.Composite)
	})
}

func TestExplicitAdditionalPropertiesWithUnknownField(t *testing.T) {
	input := []byte(`{
		"AdditionalProperties": {"existing": "value"},
		"custom": {"endpoint": "localhost"}
	}`)

	var got SpanExporter
	require.NoError(t, json.Unmarshal(input, &got))

	properties, ok := got.AdditionalProperties.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", properties["existing"])
	assert.Equal(t, map[string]any{"endpoint": "localhost"}, properties["custom"])
	assert.NotContains(t, properties, "AdditionalProperties")
}

func TestMergeAdditionalPropertiesDoesNotMutateExistingMap(t *testing.T) {
	existing := map[string]any{
		"existing": "value",
		"shared":   "original",
	}
	raw := map[string]any{
		"custom": map[string]any{"endpoint": "localhost"},
		"shared": "new",
	}

	got, ok := mergeAdditionalProperties(existing, raw).(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "value", existing["existing"])
	assert.Equal(t, "original", existing["shared"])
	assert.NotContains(t, existing, "custom")

	assert.Equal(t, "value", got["existing"])
	assert.Equal(t, "new", got["shared"])
	assert.Equal(t, map[string]any{"endpoint": "localhost"}, got["custom"])
}

func TestRepeatedJSONDecodeAdditionalProperties(t *testing.T) {
	tests := []struct {
		name string
		new  func() (any, func(any) any)
	}{
		{
			name: "SpanExporter",
			new: func() (any, func(any) any) {
				return &SpanExporter{}, func(v any) any {
					return v.(*SpanExporter).AdditionalProperties
				}
			},
		},
		{
			name: "LogRecordExporter",
			new: func() (any, func(any) any) {
				return &LogRecordExporter{}, func(v any) any {
					return v.(*LogRecordExporter).AdditionalProperties
				}
			},
		},
		{
			name: "Propagator",
			new: func() (any, func(any) any) {
				return &Propagator{}, func(v any) any {
					return v.(*Propagator).AdditionalProperties
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("replacement", func(t *testing.T) {
				target, getProperties := tt.new()

				require.NoError(t, json.Unmarshal(
					[]byte(`{"custom":{"endpoint":"old"}}`), target,
				))
				require.NoError(t, json.Unmarshal(
					[]byte(`{"custom":{"endpoint":"new"}}`), target,
				))

				properties, ok := getProperties(target).(map[string]any)
				require.True(t, ok)
				assert.Equal(t, map[string]any{"endpoint": "new"}, properties["custom"])
			})

			t.Run("null replaces old value", func(t *testing.T) {
				target, getProperties := tt.new()

				require.NoError(t, json.Unmarshal(
					[]byte(`{"custom":{"endpoint":"old"}}`), target,
				))
				require.NoError(t, json.Unmarshal(
					[]byte(`{"custom":null}`), target,
				))

				properties, ok := getProperties(target).(map[string]any)
				require.True(t, ok)
				assert.Contains(t, properties, "custom")
				assert.Nil(t, properties["custom"])
			})

			t.Run("omitted key is preserved", func(t *testing.T) {
				target, getProperties := tt.new()

				require.NoError(t, json.Unmarshal(
					[]byte(`{"custom":{"endpoint":"old"}}`), target,
				))
				require.NoError(t, json.Unmarshal([]byte(`{}`), target))

				properties, ok := getProperties(target).(map[string]any)
				require.True(t, ok)
				assert.Equal(t, map[string]any{"endpoint": "old"}, properties["custom"])
			})
		})
	}
}

func TestRepeatedYAMLDecodeExplicitAdditionalProperties(t *testing.T) {
	tests := []struct {
		name   string
		target any
	}{
		{
			name:   "SpanExporter",
			target: &SpanExporter{},
		},
		{
			name:   "LogRecordExporter",
			target: &LogRecordExporter{},
		},
		{
			name:   "Propagator",
			target: &Propagator{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode explicit properties first, then an unknown field.
			first := []byte("additionalproperties:\n  existing: value\n")
			second := []byte("custom:\n  endpoint: localhost\n")

			require.NoError(t, yaml.Unmarshal(first, tt.target))
			require.NoError(t, yaml.Unmarshal(second, tt.target))

			switch got := tt.target.(type) {
			case *SpanExporter:
				assertYamlV2Properties(t, got.AdditionalProperties)
			case *LogRecordExporter:
				assertYamlV2Properties(t, got.AdditionalProperties)
			case *Propagator:
				assertYamlV2Properties(t, got.AdditionalProperties)
			}
		})
	}
}

func assertYamlV2Properties(t *testing.T, got any) {
	t.Helper()

	properties, ok := got.(map[any]any)
	require.True(t, ok, "expected YAML v2 map[any]any, got %T", got)
	assert.Equal(t, "value", properties["existing"])
	assert.Equal(
		t,
		map[any]any{"endpoint": "localhost"},
		properties["custom"],
	)
}
