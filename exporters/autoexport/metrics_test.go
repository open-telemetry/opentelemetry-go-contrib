// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	prometheusbridge "go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	otlpmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
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

func TestMetricExporterOTLPWithDedicatedProtocol(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")

	for _, tc := range []struct {
		protocol, exporterType string
	}{
		{"http/protobuf", "*otlpmetrichttp.Exporter"},
		{"", "*otlpmetrichttp.Exporter"},
		{"grpc", "*otlpmetricgrpc.Exporter"},
	} {
		t.Run(fmt.Sprintf("protocol=%q", tc.protocol), func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", tc.protocol)

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
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
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

func TestMetricProducerPrometheusWithOTLPExporter(t *testing.T) {
	assertNoOtelHandleErrors(t)

	requestWaitChan := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, r.Body.Close())

		// Now parse the otlp proto message from request body.
		req := &otlpmetrics.ExportMetricsServiceRequest{}
		assert.NoError(t, proto.Unmarshal(body, req))

		// This is 0 without the producer registered.
		assert.NotZero(t, req.ResourceMetrics)
		assert.NotZero(t, req.ResourceMetrics[0].ScopeMetrics)
		assert.NotZero(t, req.ResourceMetrics[0].ScopeMetrics[0].Metrics)
		close(requestWaitChan)
	}))

	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", ts.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_METRICS_PRODUCERS", "prometheus")

	r, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	assert.IsType(t, &metric.PeriodicReader{}, r)

	// Register it with a meter provider to ensure it is used.
	// mp.Shutdown errors out because r.Shutdown closes the reader.
	metric.NewMeterProvider(metric.WithReader(r))

	// Shutdown actually makes an export call.
	assert.NoError(t, r.Shutdown(context.Background()))

	<-requestWaitChan
	ts.Close()
	goleak.VerifyNone(t)
}

func TestMetricProducerPrometheusWithPrometheusExporter(t *testing.T) {
	assertNoOtelHandleErrors(t)

	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "0")
	t.Setenv("OTEL_METRICS_PRODUCERS", "prometheus")

	r, err := NewMetricReader(context.Background())
	assert.NoError(t, err)

	// pull-based exporters like Prometheus need to be registered
	mp := metric.NewMeterProvider(metric.WithReader(r))

	rws, ok := r.(readerWithServer)
	if !ok {
		t.Errorf("expected readerWithServer but got %v", r)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", rws.addr))
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// By default there are two metrics exporter. target_info and promhttp_metric_handler_errors_total.
	// But by including the prometheus producer we should have more.
	assert.Greater(t, strings.Count(string(body), "# HELP"), 2)

	assert.NoError(t, mp.Shutdown(context.Background()))
	goleak.VerifyNone(t)
}

func TestMetricProducerFallbackWithPrometheusExporter(t *testing.T) {
	assertNoOtelHandleErrors(t)

	reg := prometheus.NewRegistry()
	someDummyMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dummy_metric",
		Help: "dummy metric",
	})
	reg.MustRegister(someDummyMetric)

	WithFallbackMetricProducer(func(context.Context) (metric.Producer, error) {
		return prometheusbridge.NewMetricProducer(prometheusbridge.WithGatherer(reg)), nil
	})

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
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "HELP dummy_metric_total dummy metric")

	assert.NoError(t, mp.Shutdown(context.Background()))
	goleak.VerifyNone(t)
}

func TestMultipleMetricProducerWithOTLPExporter(t *testing.T) {
	requestWaitChan := make(chan struct{})

	reg1 := prometheus.NewRegistry()
	someDummyMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dummy_metric_1",
		Help: "dummy metric ONE",
	})
	reg1.MustRegister(someDummyMetric)
	reg2 := prometheus.NewRegistry()
	someOtherDummyMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dummy_metric_2",
		Help: "dummy metric TWO",
	})
	reg2.MustRegister(someOtherDummyMetric)

	RegisterMetricProducer("first_producer", func(context.Context) (metric.Producer, error) {
		return prometheusbridge.NewMetricProducer(prometheusbridge.WithGatherer(reg1)), nil
	})
	RegisterMetricProducer("second_producer", func(context.Context) (metric.Producer, error) {
		return prometheusbridge.NewMetricProducer(prometheusbridge.WithGatherer(reg2)), nil
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, r.Body.Close())

		// Now parse the otlp proto message from request body.
		req := &otlpmetrics.ExportMetricsServiceRequest{}
		assert.NoError(t, proto.Unmarshal(body, req))

		metricNames := []string{}
		sm := req.ResourceMetrics[0].ScopeMetrics

		for i := 0; i < len(sm); i++ {
			m := sm[i].Metrics
			for i := 0; i < len(m); i++ {
				metricNames = append(metricNames, m[i].Name)
			}
		}

		assert.ElementsMatch(t, metricNames, []string{"dummy_metric_1", "dummy_metric_2"})

		close(requestWaitChan)
	}))

	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", ts.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_METRICS_PRODUCERS", "first_producer,second_producer,first_producer")

	r, err := NewMetricReader(context.Background())
	assert.NoError(t, err)
	assert.IsType(t, &metric.PeriodicReader{}, r)

	// Register it with a meter provider to ensure it is used.
	// mp.Shutdown errors out because r.Shutdown closes the reader.
	metric.NewMeterProvider(metric.WithReader(r))

	// Shutdown actually makes an export call.
	assert.NoError(t, r.Shutdown(context.Background()))

	<-requestWaitChan
	ts.Close()
	goleak.VerifyNone(t)
}
