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
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestSpans(t *testing.T) {
	var fallbackExporter testSpanExporter

	signal[trace.SpanExporter]{
		newExporter: func() (trace.SpanExporter, error) {
			return NewSpanExporter(context.Background())
		},
		fallbackExporter: &fallbackExporter,
		newExporterWithFallback: func() (trace.SpanExporter, error) {
			return NewSpanExporter(context.Background(), WithFallbackSpanExporter(&fallbackExporter))
		},
		assertOTLPHTTP: assertOTLPHTTPSpanExporter,
		assertOTLPGRPC: func(t *testing.T, got trace.SpanExporter) {
			t.Helper()

			if !assert.IsType(t, &otlptrace.Exporter{}, got) {
				return
			}

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type().String()
			assert.Equal(t, "*otlptracegrpc.client", clientType)

			assert.False(t, IsNoneSpanExporter(got))
		},
		isNoneExporter: IsNoneSpanExporter,
		envVariable:    "OTEL_TRACES_EXPORTER",
	}.testAll(t)
}

func TestMetrics(t *testing.T) {
	fallbackExporter := metric.NewManualReader()

	signal[metric.Reader]{
		newExporter: func() (metric.Reader, error) {
			return NewMetricReader(context.Background())
		},
		fallbackExporter: fallbackExporter,
		newExporterWithFallback: func() (metric.Reader, error) {
			return NewMetricReader(context.Background(), WithFallbackMetricReader(fallbackExporter))
		},
		assertOTLPHTTP: assertOTLPHTTPMetricReader,
		assertOTLPGRPC: func(t *testing.T, got metric.Reader) {
			t.Helper()

			if !assert.IsType(t, metric.NewPeriodicReader(nil), got) {
				return
			}

			// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
			clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type().String()
			assert.Equal(t, "*otlpmetricgrpc.Exporter", clientType)
		},
		isNoneExporter: IsNoneMetricReader,
		envVariable:    "OTEL_METRICS_EXPORTER",
	}.testAll(t)
}

type signal[T any] struct {
	newExporter, newExporterWithFallback func() (T, error)
	fallbackExporter                     T
	assertOTLPHTTP, assertOTLPGRPC       func(t *testing.T, got T)
	isNoneExporter                       func(exporter T) bool
	envVariable                          string
}

func (s signal[T]) testAll(t *testing.T) {
	t.Run("OTLPExporterReturnedWhenNoEnvOrFallbackExporterConfigured", func(t *testing.T) {
		exporter, err := s.newExporter()
		assert.NoError(t, err)
		s.assertOTLPHTTP(t, exporter)
	})

	t.Run("FallbackExporterReturnedWhenNoEnvExporterConfigured", func(t *testing.T) {
		exporter, err := s.newExporterWithFallback()
		assert.NoError(t, err)
		assert.Equal(t, s.fallbackExporter, exporter)
		assert.False(t, s.isNoneExporter(exporter))
	})

	t.Run("EnvExporterIsPreferredOverFallbackExporter", func(t *testing.T) {
		t.Setenv(s.envVariable, "otlp")

		exporter, err := s.newExporterWithFallback()
		assert.NoError(t, err)
		s.assertOTLPHTTP(t, exporter)
	})

	t.Run("EnvExporterOTLPOverHTTP", func(t *testing.T) {
		t.Setenv(s.envVariable, "otlp")
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

		exporter, err := s.newExporter()
		assert.NoError(t, err)
		s.assertOTLPHTTP(t, exporter)
	})

	t.Run("EnvExporterOTLPOverGRPC", func(t *testing.T) {
		t.Setenv(s.envVariable, "otlp")
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

		exporter, err := s.newExporter()
		assert.NoError(t, err)
		s.assertOTLPGRPC(t, exporter)
	})

	t.Run("EnvExporterOTLPOverGRPCOnlyProtocol", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

		exporter, err := s.newExporter()
		assert.NoError(t, err)
		s.assertOTLPGRPC(t, exporter)
	})

	t.Run("EnvExporterOTLPInvalidProtocol", func(t *testing.T) {
		t.Setenv(s.envVariable, "otlp")
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "invalid")

		exporter, err := s.newExporter()
		assert.Error(t, err)
		assert.Nil(t, exporter)
	})

	t.Run("EnvExporterNone", func(t *testing.T) {
		t.Setenv(s.envVariable, "none")

		exporter, err := s.newExporter()
		assert.NoError(t, err)
		assert.True(t, s.isNoneExporter(exporter))
	})
}

func assertOTLPHTTPMetricReader(t *testing.T, got metric.Reader) {
	t.Helper()

	if !assert.IsType(t, metric.NewPeriodicReader(nil), got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type().String()
	assert.Equal(t, "*otlpmetrichttp.Exporter", clientType)
}

func assertOTLPHTTPSpanExporter(t *testing.T, got trace.SpanExporter) {
	t.Helper()

	if !assert.IsType(t, &otlptrace.Exporter{}, got) {
		return
	}

	// Implementation detail hack. This may break when bumping OTLP exporter modules as it uses unexported API.
	clientType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("client").Elem().Type().String()
	assert.Equal(t, "*otlptracehttp.client", clientType)

	assert.False(t, IsNoneSpanExporter(got))
}

type testSpanExporter struct{}

func (e *testSpanExporter) ExportSpans(ctx context.Context, ss []trace.ReadOnlySpan) error {
	return nil
}

func (e *testSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}
