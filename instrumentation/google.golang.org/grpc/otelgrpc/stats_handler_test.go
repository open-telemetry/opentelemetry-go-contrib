// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
)

func TestNilInstruments(t *testing.T) {
	mp := meterProvider{}
	opts := []Option{WithMeterProvider(mp)}

	ctx := context.Background()

	t.Run("ServerHandler", func(t *testing.T) {
		hIface := NewServerHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*serverHandler)(nil), hIface)

		h := hIface.(*serverHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0) }, "duration")
		assert.NotPanics(t, func() { h.inSize.Record(ctx, 0) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.Record(ctx, 0) }, "outSize")
		assert.NotPanics(t, func() { h.inMsg.Record(ctx, 0) }, "inMsg")
		assert.NotPanics(t, func() { h.outMsg.Record(ctx, 0) }, "outMsg")
	})

	t.Run("ClientHandler", func(t *testing.T) {
		hIface := NewClientHandler(opts...)
		require.NotNil(t, hIface, "handler")
		require.IsType(t, (*clientHandler)(nil), hIface)

		h := hIface.(*clientHandler)

		assert.NotPanics(t, func() { h.duration.Record(ctx, 0) }, "duration")
		assert.NotPanics(t, func() { h.inSize.Record(ctx, 0) }, "inSize")
		assert.NotPanics(t, func() { h.outSize.Record(ctx, 0) }, "outSize")
		assert.NotPanics(t, func() { h.inMsg.Record(ctx, 0) }, "inMsg")
		assert.NotPanics(t, func() { h.outMsg.Record(ctx, 0) }, "outMsg")
	})
}

type meterProvider struct {
	embedded.MeterProvider
}

func (meterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return meter{}
}

type meter struct {
	// Panic for non-implemented methods.
	metric.Meter
}

func (meter) Int64Histogram(string, ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	return nil, assert.AnError
}

func (meter) Float64Histogram(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return nil, assert.AnError
}
