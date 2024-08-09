// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func TestLogExporterNone(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	exporters, err := NewLogExporters(context.Background())
	assert.NoError(t, err)
	got := exporters[0]
	t.Cleanup(func() {
		assert.NoError(t, got.ForceFlush(context.Background()))
		assert.NoError(t, got.Shutdown(context.Background()))
	})
	assert.NoError(t, got.Export(context.Background(), nil))
	assert.True(t, IsNoneLogExporter(got))
}

func TestLogExporterConsole(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "console")
	exporters, err := NewLogExporters(context.Background())
	assert.NoError(t, err)

	got := exporters[0]
	assert.IsType(t, &stdoutlog.Exporter{}, got)
}

func TestLogExporterOTLP(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, clientType string
	}{
		{"http/protobuf", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
		{"", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", tc.protocol)

			exporters, err := NewLogExporters(context.Background())
			assert.NoError(t, err)
			got := exporters[0]
			t.Cleanup(func() {
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
		{"", "atomic.Pointer[go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp.client]"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", tc.protocol)

			exporters, err := NewLogExporters(context.Background())
			got := exporters[0]
			assert.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, got.Shutdown(context.Background()))
			})
			assert.Implements(t, new(log.Exporter), got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Type()
			assert.Equal(t, tc.clientType, clientType.String())
		})
	}
}

func TestLogExporterOTLPMultiple(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp,console")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	exporters, err := NewLogExporters(context.Background())
	assert.NoError(t, err)
	assert.Len(t, exporters, 2)

	assert.Implements(t, new(log.Exporter), exporters[0])
	assert.IsType(t, &otlploghttp.Exporter{}, exporters[0])

	assert.Implements(t, new(log.Exporter), exporters[1])
	assert.IsType(t, &stdoutlog.Exporter{}, exporters[1])

	t.Cleanup(func() {
		assert.NoError(t, exporters[0].Shutdown(context.Background()))
		assert.NoError(t, exporters[1].Shutdown(context.Background()))
	})
}

func TestLogExporterOTLPMultiple_FailsIfOneValueIsInvalid(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp,something")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	_, err := NewLogExporters(context.Background())
	assert.Error(t, err)
}

func TestLogExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewLogExporters(context.Background())
	assert.Error(t, err)
}

func TestLogExporterDeprecatedNewLogExporterReturnsTheFirstExporter(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "console,otlp")
	got, err := NewLogExporter(context.Background())

	assert.NoError(t, err)
	assert.IsType(t, &stdoutlog.Exporter{}, got)
}
