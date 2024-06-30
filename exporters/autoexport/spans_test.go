// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"github.com/stretchr/testify/assert"
)

func TestSpanExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	exporters, err := NewSpanExporters(context.Background())
	got := exporters[0]
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(context.Background()))
	})
	assert.True(t, IsNoneSpanExporter(got))
}

func TestSpanExporterConsole(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "console")
	exporters, err := NewSpanExporters(context.Background())
	assert.NoError(t, err)

	got := exporters[0]
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

			exporters, err := NewSpanExporters(context.Background())
			assert.NoError(t, err)
			got := exporters[0]
			t.Cleanup(func() {
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

			exporters, err := NewSpanExporters(context.Background())
			got := exporters[0]
			assert.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.IsType(t, &otlptrace.Exporter{}, got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestSpanExporterOTLPMultiple(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp,console")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	exporters, err := NewSpanExporters(context.Background())
	assert.NoError(t, err)
	assert.Len(t, exporters, 2)

	assert.Implements(t, new(trace.SpanExporter), exporters[0])
	assert.IsType(t, &otlptrace.Exporter{}, exporters[0])

	assert.Implements(t, new(trace.SpanExporter), exporters[1])
	assert.IsType(t, &stdouttrace.Exporter{}, exporters[1])

	t.Cleanup(func() {
		assert.NoError(t, exporters[0].Shutdown(context.Background()))
		assert.NoError(t, exporters[1].Shutdown(context.Background()))
	})
}

func TestSpanExporterOTLPMultiple_FailsIfOneValueIsInvalid(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp,something")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	_, err := NewSpanExporters(context.Background())
	assert.Error(t, err)
}

func TestSpanExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewSpanExporter(context.Background())
	assert.Error(t, err)
}

func TestSpanExporterDeprecatedNewSpanExporterReturnsTheFirstExporter(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "console,otlp")
	got, err := NewSpanExporter(context.Background())

	assert.NoError(t, err)
	assert.IsType(t, &stdouttrace.Exporter{}, got)
}
