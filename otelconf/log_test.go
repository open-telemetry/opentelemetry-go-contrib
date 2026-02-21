// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

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
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdklogtest "go.opentelemetry.io/otel/sdk/log/logtest"
	"go.opentelemetry.io/otel/sdk/resource"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
			wantErr:      newErrInvalid("must not specify multiple log processor type"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp, shutdown, err := loggerProvider(tt.cfg, resource.Default())
			require.Equal(t, tt.wantProvider, mp)
			assert.ErrorIs(t, err, tt.wantErr)
			require.NoError(t, shutdown(t.Context()))
		})
	}
}

func TestLogProcessor(t *testing.T) {
	ctx := t.Context()

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
		wantErrT      error
		wantProcessor sdklog.Processor
	}{
		{
			name:     "no processor",
			wantErrT: newErrInvalid("unsupported log processor type, must be one of simple or batch"),
		},
		{
			name: "multiple processor types",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
				Simple: &SimpleLogRecordProcessor{},
			},
			wantErrT: newErrInvalid("must not specify multiple log processor type"),
		},
		{
			name: "batch processor invalid batch size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(0),
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{},
					},
				},
			},
			wantErrT: newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name: "batch processor invalid export timeout otlphttp exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ExportTimeout: ptr(-2),
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{},
					},
				},
			},
			wantErrT: newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name: "batch processor invalid queue size otlphttp exporter",

			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxQueueSize: ptr(-3),
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{},
					},
				},
			},
			wantErrT: newErrGreaterThanZero("max_queue_size"),
		},
		{
			name: "batch processor invalid schedule delay console exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					ScheduleDelay: ptr(-4),
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{},
					},
				},
			},
			wantErrT: newErrGreaterOrEqualZero("schedule_delay"),
		},
		{
			name: "batch processor invalid exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErrT: newErrInvalid("no valid log exporter"),
		},
		{
			name: "batch/console",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantProcessor: sdklog.NewBatchProcessor(consoleExporter),
		},
		{
			name: "batch/otlp-grpc-exporter-no-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-exporter-socket-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-good-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(filepath.Join("testdata", "ca.crt")),
							},
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
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								CaFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-grpc-bad-headerslist",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			name: "batch/otlp-grpc-bad-client-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPGrpc: &OTLPGrpcExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &GrpcTls{
								KeyFile:  ptr(filepath.Join("testdata", "bad_cert.crt")),
								CertFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-grpc-exporter-no-scheme",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpGRPCExporter),
		},
		{
			name: "batch/otlp-grpc-invalid-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-good-ca-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(filepath.Join("testdata", "ca.crt")),
							},
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
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								CaFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-http-bad-client-certificate",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint:    ptr("localhost:4317"),
							Compression: ptr("gzip"),
							Timeout:     ptr(1000),
							Tls: &HttpTls{
								KeyFile:  ptr(filepath.Join("testdata", "bad_cert.crt")),
								CertFile: ptr(filepath.Join("testdata", "bad_cert.crt")),
							},
						},
					},
				},
			},
			wantErrT: newErrInvalid("tls configuration"),
		},
		{
			name: "batch/otlp-http-bad-headerslist",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-exporter-no-scheme",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-endpoint",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewBatchProcessor(otlpHTTPExporter),
		},
		{
			name: "batch/otlp-http-invalid-compression",
			processor: LogRecordProcessor{
				Batch: &BatchLogRecordProcessor{
					MaxExportBatchSize: ptr(1),
					ExportTimeout:      ptr(0),
					MaxQueueSize:       ptr(1),
					ScheduleDelay:      ptr(0),
					Exporter: LogRecordExporter{
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
			name: "simple/no-exporter",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{},
				},
			},
			wantErrT: newErrInvalid("no valid log exporter"),
		},
		{
			name: "simple/console",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console: ConsoleExporter{},
					},
				},
			},
			wantProcessor: sdklog.NewSimpleProcessor(consoleExporter),
		},
		{
			name: "simple/otlp_file",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPFileDevelopment: &ExperimentalOTLPFileExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("otlp_file/development"),
		},
		{
			name: "simple/multiple",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						Console:  ConsoleExporter{},
						OTLPGrpc: &OTLPGrpcExporter{},
					},
				},
			},
			wantErrT: newErrInvalid("must not specify multiple exporters"),
		},
		{
			name: "simple/otlp-exporter",
			processor: LogRecordProcessor{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
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
			wantProcessor: sdklog.NewSimpleProcessor(otlpHTTPExporter),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logProcessor(t.Context(), tt.processor)
			require.ErrorIs(t, err, tt.wantErrT)
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

func TestLoggerProviderOptions(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls++
	}))
	defer srv.Close()

	cfg := OpenTelemetryConfiguration{
		LoggerProvider: &LoggerProvider{
			Processors: []LogRecordProcessor{{
				Simple: &SimpleLogRecordProcessor{
					Exporter: LogRecordExporter{
						OTLPHttp: &OTLPHttpExporter{
							Endpoint: ptr(srv.URL),
						},
					},
				},
			}},
		},
	}

	var buf bytes.Buffer
	stdoutlogExporter, err := stdoutlog.New(stdoutlog.WithWriter(&buf))
	require.NoError(t, err)

	res := resource.NewSchemaless(attribute.String("foo", "bar"))
	sdk, err := NewSDK(
		WithOpenTelemetryConfiguration(cfg),
		WithLoggerProviderOptions(sdklog.WithProcessor(sdklog.NewSimpleProcessor(stdoutlogExporter))),
		WithLoggerProviderOptions(sdklog.WithResource(res)),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, sdk.Shutdown(t.Context()))
	}()

	// The exporter, which we passed in as an extra option to NewSDK,
	// should be wired up to the provider in addition to the
	// configuration-based OTLP exporter.
	logger := sdk.LoggerProvider().Logger("test")
	logger.Emit(t.Context(), log.Record{})
	assert.NotZero(t, buf)
	assert.Equal(t, 1, calls)
	// Options provided by WithMeterProviderOptions may be overridden
	// by configuration, e.g. the resource is always defined via
	// configuration.
	assert.NotContains(t, buf.String(), "foo")
}

func Test_otlpGRPCLogExporter(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// TODO (#8115): Fix the flakiness on Windows and MacOS.
		t.Skip("Test is flaky on Windows and MacOS.")
	}
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
					Timeout:     ptr(5000),
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
					Timeout:     ptr(5000),
					Tls: &GrpcTls{
						CaFile: ptr("testdata/server-certs/server.crt"),
					},
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
				ctx: t.Context(),
				otlpConfig: &OTLPGrpcExporter{
					Compression: ptr("gzip"),
					Timeout:     ptr(5000),
					Tls: &GrpcTls{
						CaFile:   ptr("testdata/server-certs/server.crt"),
						KeyFile:  ptr("testdata/client-certs/client.key"),
						CertFile: ptr("testdata/client-certs/client.crt"),
					},
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
			n, err := net.Listen("tcp4", "localhost:0")
			require.NoError(t, err)

			// We need to manually construct the endpoint using the port on which the server is listening.
			//
			// n.Addr() always returns 127.0.0.1 instead of localhost.
			// But our certificate is created with CN as 'localhost', not '127.0.0.1'.
			// So we have to manually form the endpoint as "localhost:<port>".
			_, port, err := net.SplitHostPort(n.Addr().String())
			require.NoError(t, err)
			tt.args.otlpConfig.Endpoint = ptr("localhost:" + port)

			serverOpts, err := tt.grpcServerOpts()
			require.NoError(t, err)

			startGRPCLogsCollector(t, n, serverOpts)

			exporter, err := otlpGRPCLogExporter(tt.args.ctx, tt.args.otlpConfig)
			require.NoError(t, err)

			logFactory := sdklogtest.RecordFactory{
				Body: log.StringValue("test"),
			}

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				assert.NoError(collect, exporter.Export(context.Background(), []sdklog.Record{ //nolint:usetesting // required to avoid getting a canceled context.
					logFactory.NewRecord(),
				}))
			}, 10*time.Second, 1*time.Second)
		})
	}
}

// grpcLogsCollector is an OTLP gRPC server that collects all requests it receives.
type grpcLogsCollector struct {
	collogpb.UnimplementedLogsServiceServer
}

var _ collogpb.LogsServiceServer = (*grpcLogsCollector)(nil)

// startGRPCLogsCollector returns a *grpcLogsCollector that is listening at the provided
// endpoint.
//
// If endpoint is an empty string, the returned collector will be listening on
// the localhost interface at an OS chosen port.
func startGRPCLogsCollector(t *testing.T, listener net.Listener, serverOptions []grpc.ServerOption) {
	srv := grpc.NewServer(serverOptions...)
	c := &grpcLogsCollector{}

	collogpb.RegisterLogsServiceServer(srv, c)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(listener) }()

	// Wait for the gRPC server to start accepting connections
	// to avoid race-related test flakiness.
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond)
		if !assert.NoError(collect, err, "failed to dial gRPC server") {
			return
		}
		_ = conn.Close()
	}, 10*time.Second, 1*time.Second)

	t.Cleanup(func() {
		srv.GracefulStop()
		if err := <-errCh; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			assert.NoError(t, err)
		}
	})
}

// Export handles the export req.
func (*grpcLogsCollector) Export(
	_ context.Context,
	_ *collogpb.ExportLogsServiceRequest,
) (*collogpb.ExportLogsServiceResponse, error) {
	return &collogpb.ExportLogsServiceResponse{}, nil
}
