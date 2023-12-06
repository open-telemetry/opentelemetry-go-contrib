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
	"fmt"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"github.com/stretchr/testify/assert"
)

func TestSpanExporterNone(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	got, err := NewSpanExporter(context.Background())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(context.Background()))
	})
	assert.True(t, IsNoneSpanExporter(got))
}

func TestSpanExporterConsole(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "console")
	got, err := NewSpanExporter(context.Background())
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

			got, err := NewSpanExporter(context.Background())
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

func TestSpanExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewSpanExporter(context.Background())
	assert.Error(t, err)
}
