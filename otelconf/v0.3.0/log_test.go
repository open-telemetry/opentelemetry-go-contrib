// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdklogtest "go.opentelemetry.io/otel/sdk/log/logtest"
	"go.opentelemetry.io/otel/sdk/resource"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
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

	otlpGRPCExporter, err := otlploggrpc.New(ctx)
	require.NoError(t, err)

	consoleExporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
	)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		processor     LogRecordProcessor
		args          any
		wantErr       string
		wantProcessor sdklog.Processor
	}{
		{
			name:    "no processor",
			wantErr: "unsupported log processor type, must be one of simple or batch",
		},
		{
			name: "multiple processor types",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
				Simple: &SimpleLogRecordProcessor{},
			},
			wantErr: "must not specify multiple log processor type",
		},
		{
			name: "batch processor invalid batch size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(-1),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: ptr("http/protobuf"),
						},
					},
				},
			},
			wantErr: "invalid batch size -1",
		},
		{
			name: "batch processor invalid export timeout otlphttp exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: ptr(-2),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: ptr("http/protobuf"),
						},
					},
				},
			},
			wantErr: "invalid export timeout -2",
		},
		{
			name: "batch processor invalid queue size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxQueueSize: ptr(-3),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: ptr("http/protobuf"),
						},
					},
				},
			},
			wantErr: "invalid queue size -3",
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ScheduleDelay: ptr(-4),
					Exporter: LogRecordExporter{
						OTLP: &OTLP{
							Protocol: ptr("http/protobuf"),
						},
					},
				},
			},
			wantErr: "invalid schedule delay -4",
		},
		{
			name: "batch processor invalid exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErr: "no valid log exporter",
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
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter-socket-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-good-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-bad-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			name: "batch/otlp-grpc-bad-headerslist",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			name: "batch/otlp-grpc-bad-client-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			name: "batch/otlp-grpc-exporter-no-scheme",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-invalid-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-good-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-bad-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
							Protocol:    ptr("invalid"),
							Endpoint:    ptr("https://10.0.0.0:443"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Headers: []NameStringValuePair{
								{Name: "test", Value: ptr("test1")},
							},
						},
					},
				},
			},
			wantErr: "unsupported protocol \"invalid\"",
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(0),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErr: "no valid log exporter",
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
			wantProcessor: sdklog.NewSimpleProcessor(otlpHTTPExporter),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logProcessor(context.Background(), tt.processor)
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
				wantExporterType := reflect.Indirect(reflect.ValueOf(tt.wantProcessor)).FieldByName("exporter").Elem().Type()
				gotExporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type()
				require.Equal(t, wantExporterType.String(), gotExporterType.String())
			}
		})
	}
}

func Test_otlpGRPCLogExporter(t *testing.T) {
	port, err := findRandomPort()
	require.Nil(t, err)
	type args struct {
		ctx        context.Context
		otlpConfig *OTLP
	}
	tests := []struct {
		name    string
		args    args
		want    sdklog.Exporter
		wantErr string
	}{
		{
			name: "no TLS config",
			args: args{
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:    ptr("grpc"),
					Endpoint:    ptr(fmt.Sprintf("localhost:%d", port)),
					Compression: ptr("gzip"),
					Timeout:     ptr(1000),
					Insecure:    ptr(true),
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			want: &otlploggrpc.Exporter{},
		},
		{
			name: "with TLS config",
			args: args{
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:    ptr("grpc"),
					Endpoint:    ptr(fmt.Sprintf("localhost:%d", port)),
					Compression: ptr("gzip"),
					Timeout:     ptr(1000),
					Certificate: ptr("testdata/server-certs/server.crt"),
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			want: &otlploggrpc.Exporter{},
		},
		{
			name: "with TLS config and client key",
			args: args{
				ctx: context.Background(),
				otlpConfig: &OTLP{
					Protocol:          ptr("grpc"),
					Endpoint:          ptr(fmt.Sprintf("localhost:%d", port)),
					Compression:       ptr("gzip"),
					Timeout:           ptr(1000),
					Certificate:       ptr("testdata/server-certs/server.crt"),
					ClientKey:         ptr("testdata/client-certs/client.key"),
					ClientCertificate: ptr("testdata/client-certs/client.crt"),
					Headers: []NameStringValuePair{
						{Name: "test", Value: ptr("test1")},
					},
				},
			},
			want: &otlploggrpc.Exporter{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsMode := ""
			if tt.args.otlpConfig.Insecure == nil || !*tt.args.otlpConfig.Insecure {
				tlsMode = "TLS"
			}
			if tt.args.otlpConfig.ClientCertificate != nil && *tt.args.otlpConfig.ClientCertificate != "" {
				tlsMode = "mTLS"
			}
			col, err := newGRPCLogsCollector(*tt.args.otlpConfig.Endpoint, tlsMode)
			require.Nil(t, err)

			exporter, err := otlpGRPCLogExporter(tt.args.ctx, tt.args.otlpConfig)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			logFactory := sdklogtest.RecordFactory{
				Body: log.StringValue("test"),
			}

			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				require.NoError(collect, exporter.Export(context.Background(), []sdklog.Record{
					logFactory.NewRecord(),
				}))
			}, 10*time.Second, 1*time.Second)

			// Ensure everything is flushed.
			require.NoError(t, exporter.Shutdown(context.Background()))

			// verify the reception of the sent data item
			require.GreaterOrEqual(t, len(col.storage.data), 1)
			col.srv.Stop()
		})
	}
}

// logsStorage stores uploaded OTLP log data in their proto form.
type logsStorage struct {
	dataMu sync.Mutex
	data   []*lpb.ResourceLogs
}

// Add adds the request to the Storage.
func (s *logsStorage) Add(request *collogpb.ExportLogsServiceRequest) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	s.data = append(s.data, request.ResourceLogs...)
}

// grpcLogsCollector is an OTLP gRPC server that collects all requests it receives.
type grpcLogsCollector struct {
	collogpb.UnimplementedLogsServiceServer

	storage *logsStorage

	listener net.Listener
	srv      *grpc.Server
}

var _ collogpb.LogsServiceServer = (*grpcLogsCollector)(nil)

// newGRPCLogsCollector returns a *grpcLogsCollector that is listening at the provided
// endpoint.
//
// If endpoint is an empty string, the returned collector will be listening on
// the localhost interface at an OS chosen port.
func newGRPCLogsCollector(endpoint string, tlsMode string) (*grpcLogsCollector, error) {
	if endpoint == "" {
		endpoint = "localhost:0"
	}

	c := &grpcLogsCollector{
		storage: &logsStorage{},
	}

	var err error
	c.listener, err = net.Listen("tcp", endpoint)
	if err != nil {
		return nil, err
	}

	srv, err := createGRPCServer(tlsMode)
	if err != nil {
		return nil, err
	}

	c.srv = srv
	collogpb.RegisterLogsServiceServer(c.srv, c)
	go func() { _ = c.srv.Serve(c.listener) }()

	return c, nil
}

// Export handles the export req.
func (c *grpcLogsCollector) Export(
	_ context.Context,
	req *collogpb.ExportLogsServiceRequest,
) (*collogpb.ExportLogsServiceResponse, error) {
	c.storage.Add(req)

	return &collogpb.ExportLogsServiceResponse{}, nil
}

func createGRPCServer(tlsMode string) (*grpc.Server, error) {
	opts := []grpc.ServerOption{}

	switch tlsMode {
	case "TLS":
		tlsCreds, err := credentials.NewServerTLSFromFile("testdata/server-certs/server.crt", "testdata/server-certs/server.key")
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(tlsCreds))
	case "mTLS":
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
	default:
	}

	srv := grpc.NewServer(opts...)
	return srv, nil
}

func findRandomPort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	port := l.Addr().(*net.TCPAddr).Port

	err = l.Close()
	if err != nil {
		return 0, err
	}

	return port, nil
}
