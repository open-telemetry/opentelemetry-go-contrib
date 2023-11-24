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

func TestInitTracerPovider(t *testing.T) {
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
	}
	for _, tt := range tests {
		tp, shutdown, err := initTracerProvider(tt.cfg, resource.Default())
		require.Equal(t, tt.wantProvider, tp)
		require.NoError(t, tt.wantErr, err)
		require.NoError(t, shutdown(context.Background()))
	}
}

func TestSpanProcessor(t *testing.T) {
	consoleExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	require.NoError(t, err)
	otlpGRPCExporter, err := otlptracegrpc.New(context.TODO())
	require.NoError(t, err)
	otlpHTTPExporter, err := otlptracehttp.New(context.TODO())
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
			wantErr: errors.New("unsupported span processor type {<nil> <nil>}"),
		},
		{
			name: "batch processor invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			wantErr: errNoValidSpanExporter,
		},
		{
			name: "batch processor invalid batch size console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(-1),
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
					ExportTimeout: intPtr(-2),
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
					MaxQueueSize: intPtr(-3),
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
					ScheduleDelay: intPtr(-4),
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
			wantErr: errors.New("invalid schedule delay -4"),
		},
		{
			name: "batch processor console exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol: *strPtr("http/invalid"),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "batch/otlp-grpc-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Endpoint:    "http://localhost:4318/path/123",
							Compression: strPtr("none"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
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
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Endpoint:    " ",
							Compression: strPtr("gzip"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "batch/otlp-http-invalid-compression",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					MaxExportBatchSize: intPtr(0),
					ExportTimeout:      intPtr(0),
					MaxQueueSize:       intPtr(0),
					ScheduleDelay:      intPtr(0),
					Exporter: SpanExporter{
						OTLP: &OTLP{
							Protocol:    "http/protobuf",
							Endpoint:    "localhost:4318",
							Compression: strPtr("invalid"),
							Timeout:     intPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported compression \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initSpanProcessor(context.Background(), tt.processor)
			assert.Equal(t, tt.wantErr, err)
			if tt.wantProcessor == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantProcessor), reflect.TypeOf(got))
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantProcessor)).FieldByName("e").Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("e").Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
			}
		})
	}
}
