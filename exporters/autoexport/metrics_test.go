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

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric"
)

func TestMetricOTLPOverHTTP(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)

	if !assert.IsType(t, metric.NewPeriodicReader(nil), got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type().String()
	assert.Equal(t, "*otlpmetrichttp.Exporter", clientType)
}

func TestMetricOTLPOverGRPC(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)

	if !assert.IsType(t, metric.NewPeriodicReader(nil), got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type().String()
	assert.Equal(t, "*otlpmetricgrpc.Exporter", clientType)
}

func TestMetricOTLPOverInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	_, err := NewMetricReader(context.Background())
	assert.Error(t, err)
}
