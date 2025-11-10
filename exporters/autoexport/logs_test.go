// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func TestLogExporterNone(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	got, err := NewLogExporter(t.Context())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.ForceFlush(t.Context()))
		assert.NoError(t, got.Shutdown(t.Context()))
	})
	assert.NoError(t, got.Export(t.Context(), nil))
	assert.True(t, IsNoneLogExporter(got))
}

func TestLogExporterConsole(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "console")
	got, err := NewLogExporter(t.Context())
	assert.NoError(t, err)
	assert.IsType(t, &stdoutlog.Exporter{}, got)
}

func TestLogExporterOTLP(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, clientType string
	}{
		{"http/protobuf", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
		{"grpc", "otlploggrpc.logClient"},
		{"", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", tc.protocol)

			got, err := NewLogExporter(t.Context())
			assert.NoError(t, err)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.Implements(t, new(log.Exporter), got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestLogExporterOTLPWithDedicatedProtocol(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, clientType string
	}{
		{"http/protobuf", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
		{"grpc", "otlploggrpc.logClient"},
		{"", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", tc.protocol)

			got, err := NewLogExporter(t.Context())
			assert.NoError(t, err)
			t.Cleanup(func() {
				//nolint:usetesting // required to avoid getting a canceled context at cleanup.
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.Implements(t, new(log.Exporter), got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestLogExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewLogExporter(t.Context())
	assert.Error(t, err)
}

func TestLogExporterFallbackWithConsoleExporter(t *testing.T) {
	ctx := t.Context()

	fallbackExporterFactory := func(context.Context) (log.Exporter, error) {
		return stdoutlog.New()
	}

	t.Setenv("OTEL_LOGS_EXPORTER", "")

	got, err := NewLogExporter(ctx, WithFallbackLogExporter(fallbackExporterFactory))

	assert.NoError(t, err)
	assert.NotNil(t, got)

	assert.IsType(t, &stdoutlog.Exporter{}, got)

	assert.NoError(t, got.Shutdown(ctx))
}
