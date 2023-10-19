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

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/stretchr/testify/assert"
)

func TestMetricExporterNone(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	assert.True(t, IsNoneMetricReader(got))
}

func TestMetricExporterOTLP(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, exporterType string
	}{
		{"http/protobuf", "*otlpmetrichttp.Exporter"},
		{"", "*otlpmetrichttp.Exporter"},
		{"grpc", "*otlpmetricgrpc.Exporter"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", tc.protocol)

			got, err := NewMetricReader(context.Background())
			assert.NoError(t, err)
			assert.IsType(t, &metric.PeriodicReader{}, got)

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			exporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type()
			assert.Equal(t, tc.exporterType, exporterType.String())
		})
	}
}

func TestMetricExporterOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid-protocol")

	_, err := NewMetricReader(context.Background())
	assert.Error(t, err)
}

func TestMetricExporterPrometheus(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")

	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	assert.IsType(t, &prometheus.Exporter{}, got)
}
