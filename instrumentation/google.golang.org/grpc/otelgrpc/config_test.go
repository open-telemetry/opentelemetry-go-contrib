// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
)

func TestNilInstruments(t *testing.T) {
	mp := meterProvider{}
	c := newConfig([]Option{WithMeterProvider(mp)}, "test")

	ctx := context.Background()
	assert.NotPanics(t, func() { c.rpcDuration.Record(ctx, 0) }, "rpcDuration")
	assert.NotPanics(t, func() { c.rpcInBytes.Record(ctx, 0) }, "rpcInBytes")
	assert.NotPanics(t, func() { c.rpcOutBytes.Record(ctx, 0) }, "rpcOutBytes")
	assert.NotPanics(t, func() { c.rpcInMessages.Record(ctx, 0) }, "rpcInMessages")
	assert.NotPanics(t, func() { c.rpcOutMessages.Record(ctx, 0) }, "rpcOutMessages")
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
