// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	v1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

func TestTracerPovider(t *testing.T) {
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
			wantErr:      errors.Join(errors.New("must not specify multiple span processor type")),
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
										Console: Console{},
										OTLP:    &OTLP{},
									},
								},
							},
						},
					},
				},
			},
			wantProvider: noop.NewTracerProvider(),
			wantErr:      errors.Join(errors.New("must not specify multiple span processor type"), errors.New("must not specify multiple exporters")),
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
										Console: Console{},
									},
								},
							},
						},
						Sampler: &Sampler{},
					},
				},
			},
			wantProvider: noop.NewTracerProvider(),
			wantErr:      errors.Join(errInvalidSamplerConfiguration),
		},
	}
	for _, tt := range tests {
		tp, shutdown, err := tracerProvider(tt.cfg, resource.Default())
		require.Equal(t, tt.wantProvider, tp)
		assert.Equal(t, tt.wantErr, err)
		require.NoError(t, shutdown(context.Background()))
	}
}

func TestSpanProcessor(t *testing.T) {
	consoleExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	require.NoError(t, err)
	ctx := context.Background()
	otlpGRPCExporter, err := otlptracegrpc.New(ctx)
	require.NoError(t, err)
	otlpHTTPExporter, err := otlptracehttp.New(ctx)
	require.NoError(t, err)
	testCases := []struct {
		name          string
		processor     SpanProcessor
		args          any
		wantErr       string
		wantProcessor sdktrace.SpanProcessor
	}{
		{
			name:    "no processor",
			wantErr: "unsupported span processor type, must be one of simple or batch",
		},
		{
			name: "multiple processor types",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
				Simple: &SimpleSpanProcessor{},
			},
			wantErr: "must not specify multiple span processor type",
		},
		{
			name: "batch processor invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErr: "no valid span exporter",
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(-1),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantErr: "invalid batch size -1",
		},
		{
			name: "batch processor invalid export timeout console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ExportTimeout: ptr(-2),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantErr: "invalid export timeout -2",
		},
		{
			name: "batch processor invalid queue size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxQueueSize: ptr(-3),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantErr: "invalid queue size -3",
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					ScheduleDelay: ptr(-4),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantErr: "invalid schedule delay -4",
		},
		{
			name: "batch processor with multiple exporters",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						Console: Console{},
						OTLP:    &OTLP{},
					},
				},
			},
			wantErr: "must not specify multiple exporters",
		},
		{
			name: "batch processor console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(consoleExporter),
		},
		{
			name: "batch/otlp-exporter-invalid-protocol",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol: ptr("http/invalid"),
						},
					},
				},
			},
			wantErr: "unsupported protocol \"http/invalid\"",
		},
		{
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "ca.crt")),
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
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not create certificate authority chain from certificate",
		},
		{
			name: "batch/otlp-grpc-bad-client-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:          ptr("grpc"),
							Endpoint:          ptr("localhost:4317"),
							Compression:       ptr("gzip"),
							Timeout:           ptr(1000),
							ClientCertificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
							ClientKey:         ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not use client certificate: tls: failed to find any PEM data in certificate input",
		},
		{
			name: "batch/otlp-grpc-bad-headerslist",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErr: "invalid headers list: invalid key: \"\"",
		},
		{
			name: "batch/otlp-grpc-exporter-no-scheme",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
			wantErr: "parse \" \": invalid URI for request",
		},
		{
			name: "batch/otlp-grpc-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("grpc"),
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
			wantErr: "unsupported compression \"invalid\"",
		},
		{
			name: "batch/otlp-http-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "ca.crt")),
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
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Certificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not create certificate authority chain from certificate",
		},
		{
			name: "batch/otlp-http-bad-client-certificate",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:          ptr("http/protobuf"),
							Endpoint:          ptr("localhost:4317"),
							Compression:       ptr("gzip"),
							Timeout:           ptr(1000),
							ClientCertificate: ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
							ClientKey:         ptr(filepath.Join("..", "testdata", "bad_cert.crt")),
						},
					},
				},
			},
			wantErr: "could not use client certificate: tls: failed to find any PEM data in certificate input",
		},
		{
			name: "batch/otlp-http-bad-headerslist",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							HeadersList: ptr("==="),
						},
					},
				},
			},
			wantErr: "invalid headers list: invalid key: \"\"",
		},
		{
			name: "batch/otlp-http-exporter-with-path",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
			wantErr: "parse \" \": invalid URI for request",
		},
		{
			name: "batch/otlp-http-none-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    ptr("http/protobuf"),
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
			wantErr: "unsupported compression \"invalid\"",
		},
		{
			name: "simple/no-exporter",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErr: "no valid span exporter",
		},
		{
			name: "simple/console-exporter",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantProcessor: sdktrace.NewSimpleSpanProcessor(consoleExporter),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spanProcessor(context.Background(), tt.processor)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
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
				AlwaysOn: SamplerAlwaysOn{},
			},
			wantSampler: sdktrace.AlwaysSample(),
		},
		{
			name: "sampler configuration always off",
			sampler: &Sampler{
				AlwaysOff: SamplerAlwaysOff{},
			},
			wantSampler: sdktrace.NeverSample(),
		},
		{
			name: "sampler configuration trace ID ratio",
			sampler: &Sampler{
				TraceIDRatioBased: &SamplerTraceIDRatioBased{
					Ratio: ptr(0.54),
				},
			},
			wantSampler: sdktrace.TraceIDRatioBased(0.54),
		},
		{
			name: "sampler configuration trace ID ratio no ratio",
			sampler: &Sampler{
				TraceIDRatioBased: &SamplerTraceIDRatioBased{},
			},
			wantSampler: sdktrace.TraceIDRatioBased(1),
		},
		{
			name: "sampler configuration parent based no options",
			sampler: &Sampler{
				ParentBased: &SamplerParentBased{},
			},
			wantSampler: sdktrace.ParentBased(sdktrace.AlwaysSample()),
		},
		{
			name: "sampler configuration parent based many options",
			sampler: &Sampler{
				ParentBased: &SamplerParentBased{
					Root: &Sampler{
						AlwaysOff: SamplerAlwaysOff{},
					},
					RemoteParentNotSampled: &Sampler{
						AlwaysOn: SamplerAlwaysOn{},
					},
					RemoteParentSampled: &Sampler{
						TraceIDRatioBased: &SamplerTraceIDRatioBased{
							Ratio: ptr(0.009),
						},
					},
					LocalParentNotSampled: &Sampler{
						AlwaysOff: SamplerAlwaysOff{},
					},
					LocalParentSampled: &Sampler{
						TraceIDRatioBased: &SamplerTraceIDRatioBased{
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
				ParentBased: &SamplerParentBased{
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
	type args struct {
		ctx        context.Context
		otlpConfig *OTLP
	}
	tests := []struct {
		name           string
		args           args
		grpcServerOpts func() ([]grpc.ServerOption, error)
	}{
		{
			name: "no TLS config",
			args: args{
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:    ptr("grpc"),
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Insecure:    ptr(true),
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
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:    ptr("grpc"),
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Certificate: ptr("testdata/server-certs/server.crt"),
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				tlsCreds, err := credentials.NewServerTLSFromFile("testdata/server-certs/server.crt", "testdata/server-certs/server.key")
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
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:          ptr("grpc"),
					Compression:       ptr("gzip"),
					Timeout:           ptr(5000),
					Certificate:       ptr("testdata/server-certs/server.crt"),
					ClientKey:         ptr("testdata/client-certs/client.key"),
					ClientCertificate: ptr("testdata/client-certs/client.crt"),
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			grpcServerOpts: func() ([]grpc.ServerOption, error) {
				opts := []grpc.ServerOption{}
				cert, err := tls.LoadX509KeyPair("testdata/server-certs/server.crt", "testdata/server-certs/server.key")
				if err != nil {
					return nil, err
				}
				caCert, err := os.ReadFile("testdata/ca.crt")
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
			n, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)

			// this is a workaround, as providing 127.0.0.1 resulted in an "invalid URI for request" error
			tt.args.otlpConfig.Endpoint = ptr(strings.ReplaceAll(n.Addr().String(), "127.0.0.1", "localhost"))

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

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				assert.NoError(collect, exporter.ExportSpans(context.Background(), input.Snapshots()))
			}, 10*time.Second, 1*time.Second)
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
	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(func() {
		srv.Stop()
	})
}

// Export handles the export req.
func (c *grpcTraceCollector) Export(
	_ context.Context,
	_ *v1.ExportTraceServiceRequest,
) (*v1.ExportTraceServiceResponse, error) {
	return &v1.ExportTraceServiceResponse{}, nil
}
