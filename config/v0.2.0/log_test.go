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

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestLoggerProvider(t *testing.T) {
	tests := []struct {
		name         string
		cfg          configOptions
		wantProvider log.LoggerProvider
		wantErr      error
	}{
		{
			name:         "no-logger-provider-configured",
			wantProvider: noop.NewLoggerProvider(),
		},
		{
			name: "error-in-config",
			cfg: configOptions{
				opentelemetryConfig: OpenTelemetryConfiguration{
					LoggerProvider: &LoggerProvider{
						Processors: []LogRecordProcessor{
							{
								Simple: &SimpleLogRecordProcessor{},
								Batch:  &BatchLogRecordProcessor{},
							},
						},
					},
				},
			},
			wantProvider: noop.NewLoggerProvider(),
			wantErr:      errors.Join(errors.New("must not specify multiple log processor type")),
		},
	}
	for _, tt := range tests {
		mp, shutdown, err := loggerProvider(tt.cfg, resource.Default())
		require.Equal(t, tt.wantProvider, mp)
		assert.Equal(t, tt.wantErr, err)
		require.NoError(t, shutdown(context.Background()))
	}
}

func TestLogProcessor(t *testing.T) {
	ctx := context.Background()

	otlpHTTPExporter, err := otlploghttp.New(ctx)
	require.NoError(t, err)

	consoleExporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
	)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		processor     LogRecordProcessor
		args          any
		wantErr       error
		wantProcessor sdklog.Processor
	}{
		{
			name:    "no processor",
			wantErr: errors.New("unsupported log processor type, must be one of simple or batch"),
		},
		{
			name: "multiple processor types",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
				Simple: &SimpleLogRecordProcessor{},
			},
			wantErr: errors.New("must not specify multiple log processor type"),
		},
		{
			name: "batch processor invalid batch size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(-1),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: "http/protobuf",
						},
					},
				},
			},
			wantErr: errors.New("invalid batch size -1"),
		},
		{
			name: "batch processor invalid export timeout otlphttp exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: ptr(-2),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: "http/protobuf",
						},
					},
				},
			},
			wantErr: errors.New("invalid export timeout -2"),
		},
		{
			name: "batch processor invalid queue size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxQueueSize: ptr(-3),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: "http/protobuf",
						},
					},
				},
			},
			wantErr: errors.New("invalid queue size -3"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ScheduleDelay: ptr(-4),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: "http/protobuf",
						},
					},
				},
			},
			wantErr: errors.New("invalid schedule delay -4"),
		},
		{
			name: "batch processor invalid exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErr: errors.New("no valid log exporter"),
		},
		{
			name: "batch/console",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
						Console: map[string]any{},
					},
				},
			},
			wantProcessor: sdklog.NewBatchProcessor(consoleExporter),
		},
		{
			name: "batch/otlp-http-exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-with-path",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-scheme",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-protocol",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol:    "invalid",
							Endpoint:    "https://10.0.0.0:443",
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported protocol \"invalid\""),
		},
		{
			name: "batch/otlp-http-invalid-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-compression",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErr: errors.New("no valid log exporter"),
		},
		{
			name: "simple/console",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console: map[string]any{},
					},
				},
			},
			wantProcessor: sdklog.NewSimpleProcessor(consoleExporter),
		},
		{
			name: "simple/otlp-exporter",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewSimpleProcessor(otlpHTTPExporter),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logProcessor(context.Background(), tt.processor)
			require.Equal(t, tt.wantErr, err)
			if tt.wantProcessor == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, reflect.TypeOf(tt.wantProcessor), reflect.TypeOf(got))
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantProcessor)).FieldByName("exporter").Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
			}
		})
	}
}
