// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

func TestSpanExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	got, err := NewSpanExporter(t.Context())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(t.Context()))
	})
	assert.True(t, IsNoneSpanExporter(got))
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
