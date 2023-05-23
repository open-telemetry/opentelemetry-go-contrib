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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestOTLPExporterReturnedWhenNoEnvOrFallbackExportedConfigured(t *testing.T) {
	exporter := NewTraceExporter()
	assert.NotNil(t, exporter)
	assert.IsType(t, &otlptrace.Exporter{}, exporter)
}

func TestFallbackExporterReturnedWhenNoEnvExporterConfigured(t *testing.T) {
	testExporter := &testExporter{}
	exporter := NewTraceExporter(
		WithFallabckSpanExporter(testExporter),
	)
	assert.Equal(t, testExporter, exporter)
}

func TestEnvExporterIsPrefereredOverFallbackExporter(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")

	testExporter := &testExporter{}
	exporter := NewTraceExporter(
		WithFallabckSpanExporter(testExporter),
	)
	assert.IsType(t, &otlptrace.Exporter{}, exporter)
}

type testExporter struct{}

func (e *testExporter) ExportSpans(ctx context.Context, ss []trace.ReadOnlySpan) error {
	return nil
}

func (e *testExporter) Shutdown(ctx context.Context) error {
	return nil
}
