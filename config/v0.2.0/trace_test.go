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

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
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
		wantErr       error
		wantProcessor sdktrace.SpanProcessor
	}{
		{
			name:    "no processor",
			wantErr: errors.New("unsupported span processor type, must be one of simple or batch"),
		},
		{
			name: "multiple processor types",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
				Simple: &SimpleSpanProcessor{},
			},
			wantErr: errors.New("must not specify multiple span processor type"),
		},
		{
			name: "batch processor invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErr: errors.New("no valid span exporter"),
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
			wantErr: errors.New("invalid batch size -1"),
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
			wantErr: errors.New("invalid export timeout -2"),
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
			wantErr: errors.New("invalid queue size -3"),
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
			wantErr: errors.New("invalid schedule delay -4"),
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
			wantErr: errors.New("must not specify multiple exporters"),
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
							Protocol: "http/invalid",
						},
					},
				},
			},
			wantErr: errors.New("unsupported protocol \"http/invalid\""),
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
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpGRPCExporter),
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
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
			name: "batch/otlp-grpc-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
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
			name: "batch/otlp-http-exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
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
			wantProcessor: sdktrace.NewBatchSpanProcessor(otlpHTTPExporter),
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
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: ptr("none"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
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
			name: "batch/otlp-http-none-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
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
			name: "simple/no-exporter",
			processor: SpanProcessor{
				Simple: &SimpleSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErr: errors.New("no valid span exporter"),
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
			require.Equal(t, tt.wantErr, err)
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
