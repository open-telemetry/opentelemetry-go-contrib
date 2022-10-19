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

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

func TestOTLPExporterIsReturnedWhenNoEnvOrFallbackExporterIsConfigured(t *testing.T) {
	exporter := NewTraceExporter()
	assert.NotNil(t, exporter)

	otlpExp, err := SpanExporter("otlp")
	assert.Nil(t, err)
	assert.Equal(t, otlpExp, exporter)
}

func TestConfiguredExporterIsReturned(t *testing.T) {
	exp := &testExporter{}
	exporter := NewTraceExporter(
		WithFallabckSpanExporter(exp),
	)
	assert.Equal(t, exp, exporter)
}

func TestEnvExporterIsPreferredOverFallbackExporter(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")

	exp := &testExporter{}
	exporter := NewTraceExporter(
		WithFallabckSpanExporter(exp),
	)
	otlpExp, err := SpanExporter("otlp")
	assert.Nil(t, err)
	assert.Equal(t, otlpExp, exporter)
}

type testExporter struct{}

func (e *testExporter) ExportSpans(ctx context.Context, ss []tracesdk.ReadOnlySpan) error {
	return nil
}

func (e *testExporter) Shutdown(ctx context.Context) error {
	return nil
}
