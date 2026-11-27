// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestSpanExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	got, err := NewSpanExporter(t.Context())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(t.Context()))
	})
	assert.True(t, IsNoneSpanExporter(got))
	assert.NoError(t, got.ExportSpans(t.Context(), nil))
}

func TestSpanExporterConsole(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "console")
	got, err := NewSpanExporter(t.Context())
	assert.NoError(t, err)
	assert.IsType(t, &stdouttrace.Exporter{}, got)
}

func TestSpanExporterOTLP(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, clientType string
	}{
		{"http/protobuf", "*otlptracehttp.client"},
		{"", "*otlptracehttp.client"},
		{"grpc", "*otlptracegrpc.client"},
		{"GRPC", "*otlptracegrpc.client"},
		{"HTTP/Protobuf", "*otlptracehttp.client"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", tc.protocol)

			got, err := NewSpanExporter(t.Context())
			assert.NoError(t, err)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.IsType(t, &otlptrace.Exporter{}, got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestSpanExporterOTLPWithDedicatedProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, clientType string
	}{
		{"http/protobuf", "*otlptracehttp.client"},
		{"", "*otlptracehttp.client"},
		{"grpc", "*otlptracegrpc.client"},
		{"GRPC", "*otlptracegrpc.client"},
		{"HTTP/Protobuf", "*otlptracehttp.client"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", tc.protocol)

			got, err := NewSpanExporter(t.Context())
			assert.NoError(t, err)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.IsType(t, &otlptrace.Exporter{}, got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestSpanExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewSpanExporter(t.Context())
	assert.Error(t, err)
}

func TestSpanExporterFallback(t *testing.T) {
	testErr := errors.New("factory failed")

	testCases := []struct {
		name        string
		factory     func(context.Context) (trace.SpanExporter, error)
		expectedErr error
		verify      func(*testing.T, trace.SpanExporter)
	}{
		{
			name: "success with console exporter",
			factory: func(context.Context) (trace.SpanExporter, error) {
				return stdouttrace.New()
			},
			verify: func(t *testing.T, got trace.SpanExporter) {
				assert.NotNil(t, got)
				assert.IsType(t, &stdouttrace.Exporter{}, got)
				assert.NoError(t, got.Shutdown(t.Context()))
			},
		},
		{
			name: "error propagation",
			factory: func(context.Context) (trace.SpanExporter, error) {
				return nil, testErr
			},
			expectedErr: testErr,
			verify: func(t *testing.T, got trace.SpanExporter) {
				assert.Nil(t, got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", "")

			got, err := NewSpanExporter(t.Context(), WithFallbackSpanExporter(tc.factory))
			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			tc.verify(t, got)
		})
	}
}

func TestSpanExporterEnvIsCaseInsensitive(t *testing.T) {
	for _, tc := range []struct {
		value string
		check func(*testing.T, trace.SpanExporter)
	}{
		{"Console", func(t *testing.T, exp trace.SpanExporter) {
			assert.IsType(t, &stdouttrace.Exporter{}, exp)
		}},
		{"NONE", func(t *testing.T, exp trace.SpanExporter) {
			assert.True(t, IsNoneSpanExporter(exp))
		}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", tc.value)
			exp, err := NewSpanExporter(t.Context())
			require.NoError(t, err)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				assert.NoError(t, exp.Shutdown(context.Background()))
			})
			tc.check(t, exp)
		})
	}
}

func TestSpanExporterOTLPEnvIsCaseInsensitive(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "OTLP")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "GRPC")

	exp, err := NewSpanExporter(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:usetesting // required to avoid getting a canceled context at cleanup.
		assert.NoError(t, exp.Shutdown(context.Background()))
	})
	assert.IsType(t, &otlptrace.Exporter{}, exp)
}
