// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	v1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/contrib/otelconf/internal/testtls"
)

func TestTracerProvider(t *testing.T) {
	tests := []struct {
		name         string
		cfg          configOptions
		wantProvider trace.TracerProvider
		wantErr      error
	}{
		{
			name:         "no-tracer-provider-configured",
			wantProvider: noop.NewTracerProvider(),
		},
		{
			name: "error-in-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{
						Processors: []SpanProcessor{
							{
								Batch:  &BatchSpanProcessor{},
								Simple: &SimpleSpanProcessor{},
							},
						},
					},
				},
			},
			wantProvider: noop.NewTracerProvider(),
			wantErr:      newErrInvalid("must not specify multiple span processor type"),
		},
		{
			name: "multiple-errors-in-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{
						Processors: []SpanProcessor{
							{
								Batch:  &BatchSpanProcessor{},
								Simple: &SimpleSpanProcessor{},
							},
							{
								Simple: &SimpleSpanProcessor{
									Exporter: SpanExporter{
										Console:  ConsoleExporter{},
										OTLPHttp: &OTLPHttpExporter{},
									},
								},
							},
						},
					},
				},
			},
			wantProvider: noop.NewTracerProvider(),
			wantErr:      newErrInvalid("must not specify multiple exporters"),
		},
		{
			name: "invalid-sampler-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					TracerProvider: &TracerProvider{
						Processors: []SpanProcessor{
							{
								Simple: &SimpleSpanProcessor{
									Exporter: SpanExporter{
										Console: ConsoleExporter{},
									},
								},
							},
						},
						Sampler: &Sampler{},
					},
				},
			},
			wantProvider: noop.NewTracerProvider(),
			wantErr:      errInvalidSamplerConfiguration,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, shutdown, err := tracerProvider(tt.cfg, resource.Default())
			require.Equal(t, tt.wantProvider, tp)
			assert.ErrorIs(t, err, tt.wantErr)
			require.NoError(t, shutdown(t.Context()))
		})
	}
}

func TestTracerProviderOptions(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	defer srv.Close()

	cfg := OpenTelemetryConfiguration{
		TracerProvider: &TracerProvider{
			Processors: []SpanProcessor{{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint: ptr(srv.URL),
						},
					},
				},
			}},
		},
	}

	var buf bytes.Buffer
	stdouttraceExporter, err := stdouttrace.New(stdouttrace.WithWriter(&buf))
	require.NoError(t, err)

	res := resource.NewSchemaless(attribute.String("foo", "bar"))
	sdk, err := NewSDK(
		WithOpenTelemetryConfiguration(cfg),
		WithTracerProviderOptions(sdktrace.WithSyncer(stdouttraceExporter)),
		WithTracerProviderOptions(sdktrace.WithResource(res)),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, sdk.Shutdown(t.Context()))
	}()

	// The exporter, which we passed in as an extra option to NewSDK,
	// should be wired up to the provider in addition to the
	// configuration-based OTLP exporter.
	tracer := sdk.TracerProvider().Tracer("test")
	_, span := tracer.Start(t.Context(), "span")
	span.End()
	assert.NotZero(t, buf)
	assert.Equal(t, 1, calls)
	// Options provided by WithMeterProviderOptions may be overridden
	// by configuration, e.g. the resource is always defined via
	// configuration.
	assert.NotContains(t, buf.String(), "foo")
}

func TestSpanProcessor(t *testing.T) {
	material := testtls.Write(t)
	consoleExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := t.Context()
	otlpGRPCExporter, err := otlptracegrpc.New(ctx)
	require.NoError(t, err)
	otlpHTTPExporter, err := otlptracehttp.New(ctx)
	require.NoError(t, err)
	testCases := []struct {
		name          string
		processor     SpanProcessor
		args          any
		wantErrT      error
		wantProcessor sdktrace.SpanProcessor
	}{
		{
			name:     "no processor",
			wantErrT: newErrInvalid("unsupported span processor type, must be one of simple or batch"),
		},
		{
			name: "multiple processor types",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
				Simple: &SimpleSpanProcessor{},
			},
			wantErrT: newErrInvalid("must not specify multiple span processor type"),
		},
		{
			name: "batch processor invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErrT: newErrInvalid("no valid span exporter"),
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(-1),
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantErrT: newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name: "batch processor invalid export timeout console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ExportTimeout: ptr(-2),
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantErrT: newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name: "batch processor invalid queue size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxQueueSize: ptr(-3),
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantErrT: newErrGreaterThanZero("max_queue_size"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ScheduleDelay: ptr(-4),
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantErrT: newErrGreaterOrEqualZero("schedule_delay"),
		},
		{
			name: "batch processor with multiple exporters",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						Console:  ConsoleExporter{},
						OTLPHttp: &OTLPHttpExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("must not specify multiple exporters"),
		},
		{
			name: "batch processor console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(consoleExporter),
		},
		{
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("http://localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter-socket-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
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
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-good-ca-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(material.CACertPath),
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-bad-ca-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-grpc-bad-client-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								KeyFile:  ptr(material.BadCertPath),
								CertFile: ptr(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-grpc-bad-headerslist",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErrT: newErrInvalid("invalid headers_list"),
		},
		{
			name: "batch/otlp-grpc-exporter-no-scheme",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-invalid-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
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
			wantErrT: newErrInvalid("endpoint parsing failed"),
		},
		{
			name: "batch/otlp-grpc-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("invalid"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported compression \"invalid\""),
		},
		{
			name: "batch/otlp-http-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
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
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-good-ca-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(material.CACertPath),
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-bad-ca-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-http-bad-client-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								KeyFile:  ptr(material.BadCertPath),
								CertFile: ptr(material.BadCertPath),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-http-bad-headerslist",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErrT: newErrInvalid("invalid headers_list"),
		},
		{
			name: "batch/otlp-http-exporter-with-path",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("http://localhost:4318/path/123"),
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-scheme",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
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
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
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
			wantErrT: newErrInvalid("endpoint parsing failed"),
		},
		{
			name: "batch/otlp-http-none-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
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
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
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
			wantErrT: newErrInvalid("unsupported compression \"invalid\""),
		},
		{
			name: "batch/otlp-http-invalid-encoding",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint: ptr("http://localhost:4318"),
							Encoding: ptr(OTLPHttpEncoding("json")),
						},
					},
				},
			},
			wantErrT: newErrInvalid("unsupported encoding \"json\""),
		},
		{
			name: "simple/no-exporter",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErrT: newErrInvalid("no valid span exporter"),
		},
		{
			name: "simple/console-exporter",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantProcessor: sdktrace.NewSimpleSpanProcessor(consoleExporter),
		},
		{
			name: "simple/otlp_file",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("otlp_file/development"),
		},
		{
			name: "simple/multiple",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						Console:  ConsoleExporter{},
						OTLPGrpc: &OTLPGrpcExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("must not specify multiple exporters"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spanProcessor(t.Context(), tt.processor)
			require.ErrorIs(t, err, tt.wantErrT)
			if tt.wantProcessor == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantProcessor), reflect.TypeOf(got))
				var fieldName string
				switch reflect.TypeOf(tt.wantProcessor).String() {
				case "*trace.simpleSpanProcessor":
					fieldName = "exporter"
				default:
					fieldName = "e"
				}
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantProcessor)).FieldByName(fieldName).Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName(fieldName).Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
			}
		})
	}
}

func TestSampler(t *testing.T) {
	for _, tt := range []struct {
		name        string
		sampler     *Sampler
		wantSampler sdktrace.Sampler
		wantError   error
	}{
		{
			name:        "no sampler configuration, return default",
			sampler:     nil,
			wantSampler: sdktrace.ParentBased(sdktrace.AlwaysSample()),
		},
		{
			name:        "invalid sampler configuration, return error",
			sampler:     &Sampler{},
			wantSampler: nil,
			wantError:   errInvalidSamplerConfiguration,
		},
		{
			name: "sampler configuration always on",
			sampler: &Sampler{
				AlwaysOn: AlwaysOnSampler{},
			},
			wantSampler: sdktrace.AlwaysSample(),
		},
		{
			name: "sampler configuration always off",
			sampler: &Sampler{
				AlwaysOff: AlwaysOffSampler{},
			},
			wantSampler: sdktrace.NeverSample(),
		},
		{
			name: "sampler configuration trace ID ratio",
			sampler: &Sampler{
				TraceIDRatioBased: &TraceIDRatioBasedSampler{
					Ratio: ptr(0.54),
				},
			},
			wantSampler: sdktrace.TraceIDRatioBased(0.54),
		},
		{
			name: "sampler configuration trace ID ratio no ratio",
			sampler: &Sampler{
				TraceIDRatioBased: &TraceIDRatioBasedSampler{},
			},
			wantSampler: sdktrace.TraceIDRatioBased(1),
		},
		{
			name: "sampler configuration parent based no options",
			sampler: &Sampler{
				ParentBased: &ParentBasedSampler{},
			},
			wantSampler: sdktrace.ParentBased(sdktrace.AlwaysSample()),
		},
		{
			name: "sampler configuration parent based many options",
			sampler: &Sampler{
				ParentBased: &ParentBasedSampler{
					Root: &Sampler{
						AlwaysOff: AlwaysOffSampler{},
					},
					RemoteParentNotSampled: &Sampler{
						AlwaysOn: AlwaysOnSampler{},
					},
					RemoteParentSampled: &Sampler{
						TraceIDRatioBased: &TraceIDRatioBasedSampler{
							Ratio: ptr(0.009),
						},
					},
					LocalParentNotSampled: &Sampler{
						AlwaysOff: AlwaysOffSampler{},
					},
					LocalParentSampled: &Sampler{
						TraceIDRatioBased: &TraceIDRatioBasedSampler{
							Ratio: ptr(0.05),
						},
					},
				},
			},
			wantSampler: sdktrace.ParentBased(
				sdktrace.NeverSample(),
				sdktrace.WithLocalParentNotSampled(sdktrace.NeverSample()),
				sdktrace.WithLocalParentSampled(sdktrace.TraceIDRatioBased(0.05)),
				sdktrace.WithRemoteParentNotSampled(sdktrace.AlwaysSample()),
				sdktrace.WithRemoteParentSampled(sdktrace.TraceIDRatioBased(0.009)),
			),
		},
		{
			name: "sampler configuration with many errors",
			sampler: &Sampler{
				ParentBased: &ParentBasedSampler{
					Root:                   &Sampler{},
					RemoteParentNotSampled: &Sampler{},
					RemoteParentSampled:    &Sampler{},
					LocalParentNotSampled:  &Sampler{},
					LocalParentSampled:     &Sampler{},
				},
			},
			wantError: errors.Join(
				errInvalidSamplerConfiguration,
				errInvalidSamplerConfiguration,
				errInvalidSamplerConfiguration,
				errInvalidSamplerConfiguration,
				errInvalidSamplerConfiguration,
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sampler(tt.sampler)
			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantError.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantSampler, got)
		})
	}
}

func Test_otlpGRPCTraceExporter(t *testing.T) {
	material := testtls.Write(t)
	type args struct {
		ctx        context.Context
		otlpConfig *OTLPGrpcExporter
	}
	tests := []struct {
		name           string
		args           args
		grpcServerOpts func() ([]grpc.ServerOption, error)
	}{
		{
			name: "no TLS config",
			args: args{
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcExporter{
					Compression: ptr("gzip"),
					Timeout:     ptr(50000),
					Tls: &GrpcTls{
						Insecure: ptr(true),
					},
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
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcExporter{
					Compression: ptr("gzip"),
					Timeout:     ptr(50000),
					Tls: &GrpcTls{
						CaFile: ptr(material.CACertPath),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				tlsCreds, err := credentials.NewServerTLSFromFile(material.ServerCertPath, material.ServerKeyPath)
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
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcExporter{
					Compression: ptr("gzip"),
					Timeout:     ptr(50000),
					Tls: &GrpcTls{
						CaFile:   ptr(material.CACertPath),
						KeyFile:  ptr(material.ClientKeyPath),
						CertFile: ptr(material.ClientCertPath),
					},
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				cert, err := tls.LoadX509KeyPair(material.ServerCertPath, material.ServerKeyPath)
				if err != nil {
					return nil, err
				}
				caCert, err := os.ReadFile(material.CACertPath)
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
			n, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "localhost:0")
			require.NoError(t, err)

			scheme := "https"
			if tt.args.otlpConfig.Tls != nil && tt.args.otlpConfig.Tls.Insecure != nil && *tt.args.otlpConfig.Tls.Insecure {
				scheme = "http"
			}
			tt.args.otlpConfig.Endpoint = ptr(scheme + "://" + n.Addr().String())

			serverOpts, err := tt.grpcServerOpts()
			require.NoError(t, err)

			startGRPCTraceCollector(t, n, serverOpts)

			exporter, err := otlpGRPCSpanExporter(tt.args.ctx, tt.args.otlpConfig)
			require.NoError(t, err)

			input := tracetest.SpanStubs{
				{
					Name: "test-span",
				},
			}

			assert.NoError(t, exporter.ExportSpans(t.Context(), input.Snapshots()))
		})
	}
}

// grpcTraceCollector is an OTLP gRPC server that collects all requests it receives.
type grpcTraceCollector struct {
	v1.UnimplementedTraceServiceServer
}

var _ v1.TraceServiceServer = (*grpcTraceCollector)(nil)

// startGRPCTraceCollector returns a *grpcTraceCollector that is listening at the provided
// endpoint.
//
// If endpoint is an empty string, the returned collector will be listening on
// the localhost interface at an OS chosen port.
func startGRPCTraceCollector(t *testing.T, listener net.Listener, serverOptions []grpc.ServerOption) {
	srv := grpc.NewServer(serverOptions...)
	c := &grpcTraceCollector{}

	v1.RegisterTraceServiceServer(srv, c)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(listener) }()

	t.Cleanup(func() {
		srv.Stop()
		if err := <-errCh; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			assert.NoError(t, err)
		}
	})
}

// Export handles the export req.
func (*grpcTraceCollector) Export(
	_ context.Context,
	_ *v1.ExportTraceServiceRequest,
) (*v1.ExportTraceServiceResponse, error) {
	return &v1.ExportTraceServiceResponse{}, nil
}
