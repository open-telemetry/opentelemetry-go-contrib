// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMetricExporterNone(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(context.Background()))
	})
	assert.True(t, IsNoneMetricReader(got))
}

func TestMetricExporterConsole(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "console")
	got, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, got.Shutdown(context.Background()))
	})
	assert.IsType(t, &metric.PeriodicReader{}, got)
	exporterType := reflect.Indirect(reflect.ValueOf(got)).FieldByName("exporter").Elem().Type()
	assert.Equal(t, "*stdoutmetric.exporter", exporterType.String())
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
			t.Cleanup(func() {
				assert.NoError(t, got.Shutdown(context.Background()))
			})
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

func assertNoOtelHandleErrors(t *testing.T) {
	h := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(h) })

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(cause error) {
		t.Errorf("expected to calls to otel.Handle but got %v from %s", cause, debug.Stack())
	}))
}

func TestMetricExporterPrometheus(t *testing.T) {
	assertNoOtelHandleErrors(t)

	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "0")

	r, err := NewMetricReader(context.Background())
	assert.NoError(t, err)

	// pull-based exporters like Prometheus need to be registered
	mp := metric.NewMeterProvider(metric.WithReader(r))

	rws, ok := r.(readerWithServer)
	if !ok {
		t.Errorf("expected readerWithServer but got %v", r)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", rws.addr))
	assert.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "# HELP")

	assert.NoError(t, mp.Shutdown(context.Background()))
	goleak.VerifyNone(t)
}

func TestMetricExporterPrometheusInvalidPort(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "invalid-port")

	_, err := NewMetricReader(context.Background())
	assert.ErrorContains(t, err, "binding")
}
