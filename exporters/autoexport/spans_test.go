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

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/assert"
)

func TestSpanExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	got, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assert.True(t, IsNoneSpanExporter(got))
}

func assertOTLPExporterWithClientTypeName(t *testing.T, got trace.SpanExporter, clientTypeName string) {
	t.Helper()
	assert.IsType(t, &otlptrace.Exporter{}, got)

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type().String()
	assert.Equal(t, clientTypeName, clientType)
}

func TestSpanExporterOTLPOverHTTP(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	got, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPExporterWithClientTypeName(t, got, "*otlptracehttp.client")
}

func TestSpanExporterOTLPOverGRPC(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	got, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	assertOTLPExporterWithClientTypeName(t, got, "*otlptracegrpc.client")
}

func TestSpanExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewSpanExporter(context.Background())
	assert.Error(t, err)
}
