// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.openio/contrib/config"

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpanProcessorValidate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		processor SpanProcessor
		expected  error
	}{
		{
			name: "valid span processor",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{
						Console: Console{},
					},
				},
			},
		},
		{
			name:      "invalid span processor: no processor",
			processor: SpanProcessor{},
			expected:  errUnsupportedSpanProcessorType,
		},
		{
			name: "invalid span processor: invalid exporter",
			processor: SpanProcessor{
				Batch: &BatchSpanProcessor{
					Exporter: SpanExporter{},
				},
			},
			expected: errInvalidExporterConfiguration,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.processor.Validate(), tc.expected)
		})
	}
}

func TestMetricReader(t *testing.T) {
	testCases := []struct {
		name   string
		reader MetricReader
		args   any
		err    error
	}{
		{
			name: "noreader",
			err:  errUnsupportedMetricReaderType,
		},
		{
			name: "periodic/console-exporter-with-timeout-interval",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Interval: intToPtr(10),
					Timeout:  intToPtr(5),
					Exporter: MetricExporter{
						Console: Console{},
					},
				},
			},
		},
		{
			name: "periodic/otlp-exporter-invalid-protocol",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol: *strToPtr("http/invalid"),
						},
					},
				},
			},
			err: errors.New("unsupported protocol http/invalid"),
		},
		{
			name: "periodic/otlp-grpc-exporter-no-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol:    "grpc/protobuf",
							Compression: strToPtr("gzip"),
							Timeout:     intToPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "periodic/otlp-grpc-exporter",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "http://localhost:4317",
							Compression: strToPtr("none"),
							Timeout:     intToPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "periodic/otlp-grpc-exporter-no-scheme",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strToPtr("gzip"),
							Timeout:     intToPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
		},
		{
			name: "periodic/otlp-grpc-invalid-endpoint",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    " ",
							Compression: strToPtr("gzip"),
							Timeout:     intToPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name: "periodic/otlp-grpc-invalid-compression",
			reader: MetricReader{
				Periodic: &PeriodicMetricReader{
					Exporter: MetricExporter{
						Otlp: &OtlpMetric{
							Protocol:    "grpc/protobuf",
							Endpoint:    "localhost:4317",
							Compression: strToPtr("invalid"),
							Timeout:     intToPtr(1000),
							Headers: map[string]string{
								"test": "test1",
							},
						},
					},
				},
			},
			err: errors.New("unsupported compression \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.err, tt.reader.Validate())
		})
	}
}

func TestUnmarshallingAndValidate(t *testing.T) {
	type testInterface interface {
		UnmarshalJSON(b []byte) error
		Validate() error
	}
	testCases := []struct {
		name        string
		unmarshaler testInterface
		args        any
		err         error
	}{
		{
			name:        "metric-pull-invalid-exporter",
			unmarshaler: &PullMetricReader{},
			err:         errInvalidExporterConfiguration,
		},
		{
			name:        "metric-pull-no-exporter",
			unmarshaler: &PullMetricReader{},
			err:         fmt.Errorf("field exporter in PullMetricReader: required"),
		},
		{
			name:        "metric-pull-prometheus-invalid-config-no-host",
			unmarshaler: &PullMetricReader{},
			err:         fmt.Errorf("host must be specified"),
		},
		{
			name:        "metric-pull-prometheus-invalid-config-no-port",
			unmarshaler: &PullMetricReader{},
			err:         fmt.Errorf("port must be specified"),
		},
		{
			name:        "metric-pull-prometheus-exporter",
			unmarshaler: &PullMetricReader{},
		},
		{
			name:        "metric-periodic-invalid-exporter",
			unmarshaler: &PeriodicMetricReader{},
			err:         errInvalidExporterConfiguration,
		},
		{
			name:        "metric-periodic-no-exporter",
			unmarshaler: &PeriodicMetricReader{},
			err:         fmt.Errorf("field exporter in PeriodicMetricReader: required"),
		},
		{
			name:        "metric-periodic-console-exporter",
			unmarshaler: &PeriodicMetricReader{},
		},
		{
			name:        "metric-periodic-otlp-http-exporter-with-path",
			unmarshaler: &PeriodicMetricReader{},
		},
		{
			name:        "metric-periodic-otlp-http-exporter-no-endpoint",
			unmarshaler: &PeriodicMetricReader{},
			err:         fmt.Errorf("field endpoint in OtlpMetric: required"),
		},
		{
			name:        "metric-periodic-otlp-http-exporter-no-scheme",
			unmarshaler: &PeriodicMetricReader{},
		},
		{
			name:        "metric-periodic-otlp-http-invalid-endpoint",
			unmarshaler: &PeriodicMetricReader{},
			err:         &url.Error{Op: "parse", URL: "http:// ", Err: url.InvalidHostError(" ")},
		},
		{
			name:        "metric-periodic-otlp-http-invalid-compression",
			unmarshaler: &PeriodicMetricReader{},
			err:         fmt.Errorf("unsupported compression \"invalid\""),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := os.ReadFile(filepath.Join("testdata", fmt.Sprintf("%s.json", tt.name)))
			require.NoError(t, err)

			if err := tt.unmarshaler.UnmarshalJSON(bytes); err != nil {
				require.Equal(t, tt.err, err)
				return
			}

			assert.Equal(t, tt.err, tt.unmarshaler.Validate())
		})
	}
}
