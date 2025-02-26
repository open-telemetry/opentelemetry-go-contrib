// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
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
				WithContext(context.Background()),
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
				WithContext(context.Background()),
				WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
					Disabled:       ptr(true),
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
		require.Equal(t, tt.wantShutdownErr, sdk.Shutdown(context.Background()))
	}
}

var v03OpenTelemetryConfig = OpenTelemetryConfiguration{
	Disabled:   ptr(false),
	FileFormat: ptr("0.3"),
	AttributeLimits: &AttributeLimits{
		AttributeCountLimit:       ptr(128),
		AttributeValueLengthLimit: ptr(4096),
	},
	Instrumentation: &Instrumentation{
		Cpp: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Dotnet: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Erlang: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
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
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Java: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Js: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Php: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Python: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Ruby: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Rust: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
		Swift: LanguageSpecificInstrumentation{
			"example": map[string]interface{}{
				"property": "value",
			},
		},
	},
	LoggerProvider: &LoggerProvider{
		Limits: &LogRecordLimits{
			AttributeCountLimit:       ptr(128),
			AttributeValueLengthLimit: ptr(4096),
		},
		Processors: []LogRecordProcessor{
			{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: ptr(30000),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Certificate:       ptr("/app/cert.pem"),
							ClientCertificate: ptr("/app/cert.pem"),
							ClientKey:         ptr("/app/cert.pem"),
							Compression:       ptr("gzip"),
							Endpoint:          ptr("http://localhost:4318/v1/logs"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Insecure:    ptr(false),
							Protocol:    ptr("http/protobuf"),
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
				Producers: []MetricProducer{
					{},
				},
				Periodic: &PeriodicMetricReader{
					Exporter: PushMetricExporter{
						OTLP: &OTLPMetric{
							Certificate:                 ptr("/app/cert.pem"),
							ClientCertificate:           ptr("/app/cert.pem"),
							ClientKey:                   ptr("/app/cert.pem"),
							Compression:                 ptr("gzip"),
							DefaultHistogramAggregation: ptr(OTLPMetricDefaultHistogramAggregationBase2ExponentialBucketHistogram),
							Endpoint:                    ptr("http://localhost:4318/v1/metrics"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList:           ptr("api-key=1234"),
							Insecure:              ptr(false),
							Protocol:              ptr("http/protobuf"),
							TemporalityPreference: ptr("delta"),
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
						Console: Console{},
					},
				},
			},
		},
		Views: []View{
			{
				Selector: &ViewSelector{
					InstrumentName: ptr("my-instrument"),
					InstrumentType: ptr(ViewSelectorInstrumentTypeHistogram),
					MeterName:      ptr("my-meter"),
					MeterSchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
					MeterVersion:   ptr("1.0.0"),
					Unit:           ptr("ms"),
				},
				Stream: &ViewStream{
					Aggregation: &ViewStreamAggregation{
						ExplicitBucketHistogram: &ViewStreamAggregationExplicitBucketHistogram{
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
	Propagator: &Propagator{
		Composite: []*string{ptr("tracecontext"), ptr("baggage"), ptr("b3"), ptr("b3multi"), ptr("jaeger"), ptr("xray"), ptr("ottrace")},
	},
	Resource: &Resource{
		Attributes: []AttributeNameValue{
			{Name: "service.name", Value: "unknown_service"},
			{Name: "string_key", Type: &AttributeNameValueType{Value: "string"}, Value: "value"},
			{Name: "bool_key", Type: &AttributeNameValueType{Value: "bool"}, Value: true},
			{Name: "int_key", Type: &AttributeNameValueType{Value: "int"}, Value: 1},
			{Name: "double_key", Type: &AttributeNameValueType{Value: "double"}, Value: 1.1},
			{Name: "string_array_key", Type: &AttributeNameValueType{Value: "string_array"}, Value: []interface{}{"value1", "value2"}},
			{Name: "bool_array_key", Type: &AttributeNameValueType{Value: "bool_array"}, Value: []interface{}{true, false}},
			{Name: "int_array_key", Type: &AttributeNameValueType{Value: "int_array"}, Value: []interface{}{1, 2}},
			{Name: "double_array_key", Type: &AttributeNameValueType{Value: "double_array"}, Value: []interface{}{1.1, 2.2}},
		},
		AttributesList: ptr("service.namespace=my-namespace,service.version=1.0.0"),
		Detectors: &Detectors{
			Attributes: &DetectorsAttributes{
				Excluded: []string{"process.command_args"},
				Included: []string{"process.*"},
			},
		},
		SchemaUrl: ptr("https://opentelemetry.io/schemas/1.16.0"),
	},
	TracerProvider: &TracerProvider{
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
						OTLP: &OTLP{
							Certificate:       ptr("/app/cert.pem"),
							ClientCertificate: ptr("/app/cert.pem"),
							ClientKey:         ptr("/app/cert.pem"),
							Compression:       ptr("gzip"),
							Endpoint:          ptr("http://localhost:4318/v1/traces"),
							Headers: []NameStringValuePair{
								{Name: "api-key", Value: ptr("1234")},
							},
							HeadersList: ptr("api-key=1234"),
							Insecure:    ptr(false),
							Protocol:    ptr("http/protobuf"),
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
						Zipkin: &Zipkin{
							Endpoint: ptr("http://localhost:9411/api/v2/spans"),
							Timeout:  ptr(10000),
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
		wantType interface{}
	}{
		{
			name:    "valid YAML config",
			input:   `valid_empty.yaml`,
			wantErr: nil,
			wantType: &OpenTelemetryConfiguration{
				Disabled:   ptr(false),
				FileFormat: ptr("0.1"),
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
  line 81: cannot unmarshal !!map into []config.NameStringValuePair
  line 185: cannot unmarshal !!map into []config.NameStringValuePair
  line 244: cannot unmarshal !!seq into config.IncludeExclude
  line 305: cannot unmarshal !!map into []config.NameStringValuePair
  line 408: cannot unmarshal !!map into []config.AttributeNameValue`),
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

func TestSerializeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  error
		wantType interface{}
	}{
		{
			name:    "valid JSON config",
			input:   `valid_empty.json`,
			wantErr: nil,
			wantType: OpenTelemetryConfiguration{
				Disabled:   ptr(false),
				FileFormat: ptr("0.1"),
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

func ptr[T any](v T) *T {
	return &v
}
