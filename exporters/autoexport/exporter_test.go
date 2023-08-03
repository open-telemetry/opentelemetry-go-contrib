// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autoexport

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestOTLPExporterReturnedWhenNoEnvOrFallbackExporterConfigured(t *testing.T) {
	exporter, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPHTTPExporter(t, exporter)
}

func TestFallbackExporterReturnedWhenNoEnvExporterConfigured(t *testing.T) {
	testExporter := &testExporter{}
	exporter, err := NewSpanExporter(
		context.Background(),
		WithFallbackSpanExporter(testExporter),
	)
	assert.NoError(t, err)
	assert.Equal(t, testExporter, exporter)
	assert.False(t, IsNoneSpanExporter(exporter))
}

func TestEnvExporterIsPreferredOverFallbackExporter(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")

	testExporter := &testExporter{}
	exporter, err := NewSpanExporter(
		context.Background(),
		WithFallbackSpanExporter(testExporter),
	)
	assert.NoError(t, err)
	assertOTLPHTTPExporter(t, exporter)
}

func TestEnvExporterOTLPOverHTTP(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	exporter, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPHTTPExporter(t, exporter)
}

func TestEnvExporterOTLPOverGRPC(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	exporter, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPGRPCExporter(t, exporter)
}

func TestEnvExporterOTLPOverGRPCOnlyProtocol(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	exporter, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPGRPCExporter(t, exporter)
}

func TestEnvExporterOTLPInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid")

	exporter, err := NewSpanExporter(context.Background())
	assert.Error(t, err)
	assert.Nil(t, exporter)
}

func TestEnvExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")

	exporter, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assert.True(t, IsNoneSpanExporter(exporter))
}

func assertOTLPHTTPExporter(t *testing.T, got trace.SpanExporter) {
	t.Helper()

	if !assert.IsType(t, &otlptrace.Exporter{}, got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type().String()
	assert.Equal(t, "*otlptracehttp.client", clientType)

	assert.False(t, IsNoneSpanExporter(got))
}

func assertOTLPGRPCExporter(t *testing.T, got trace.SpanExporter) {
	t.Helper()

	if !assert.IsType(t, &otlptrace.Exporter{}, got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type().String()
	assert.Equal(t, "*otlptracegrpc.client", clientType)

	assert.False(t, IsNoneSpanExporter(got))
}

type testExporter struct{}

func (e *testExporter) ExportSpans(ctx context.Context, ss []trace.ReadOnlySpan) error {
	return nil
}

func (e *testExporter) Shutdown(ctx context.Context) error {
	return nil
}
