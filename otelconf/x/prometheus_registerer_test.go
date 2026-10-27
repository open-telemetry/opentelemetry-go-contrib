// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithPrometheusRegisterer(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := OpenTelemetryConfiguration{
		MeterProvider: &MeterProvider{
			Readers: []MetricReader{
				{
					Pull: &PullMetricReader{
						Exporter: PullMetricExporter{
							// No host or port: they are not needed with a registerer.
							PrometheusDevelopment: &ExperimentalPrometheusMetricExporter{},
						},
					},
				},
			},
		},
	}

	sdk, err := NewSDK(
		WithContext(t.Context()),
		WithOpenTelemetryConfiguration(cfg),
		WithPrometheusRegisterer(reg),
	)
	require.NoError(t, err)
	defer func() { assert.NoError(t, sdk.Shutdown(t.Context())) }()

	counter, err := sdk.MeterProvider().Meter("test").Int64Counter("registerer_test_requests")
	require.NoError(t, err)
	counter.Add(t.Context(), 3)

	families, err := reg.Gather()
	require.NoError(t, err)
	var found bool
	for _, f := range families {
		if f.GetName() == "registerer_test_requests_total" {
			found = true
			require.Len(t, f.GetMetric(), 1)
			assert.InDelta(t, 3.0, f.GetMetric()[0].GetCounter().GetValue(), 0)
		}
	}
	assert.True(t, found, "counter not registered with the provided registerer")
}

func TestPrometheusReaderWithRegistererNeedsNoListener(t *testing.T) {
	reader, err := prometheusReader(t.Context(), &ExperimentalPrometheusMetricExporter{}, prometheus.NewRegistry())
	require.NoError(t, err)
	_, isServer := reader.(readerWithServer)
	assert.False(t, isServer)
	require.NoError(t, reader.Shutdown(t.Context()))
}

func TestPrometheusReaderWithRegistererInvalidConfig(t *testing.T) {
	strategy := ExperimentalPrometheusTranslationStrategy("invalid")
	_, err := prometheusReader(t.Context(), &ExperimentalPrometheusMetricExporter{
		TranslationStrategy: &strategy,
	}, prometheus.NewRegistry())
	assert.ErrorIs(t, err, &errInvalid{})
}

// failingRegisterer rejects every collector.
type failingRegisterer struct{ prometheus.Registerer }

func (failingRegisterer) Register(prometheus.Collector) error { return errors.New("register failed") }

func TestPrometheusReaderWithRegistererRegisterError(t *testing.T) {
	_, err := prometheusReader(t.Context(), &ExperimentalPrometheusMetricExporter{}, failingRegisterer{})
	assert.ErrorContains(t, err, "register failed")
}
