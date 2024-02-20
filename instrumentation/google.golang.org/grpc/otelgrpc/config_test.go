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
	assert.NotPanics(t, func() { c.rpcRequestSize.Record(ctx, 0) }, "rpcRequestSize")
	assert.NotPanics(t, func() { c.rpcResponseSize.Record(ctx, 0) }, "rpcResponseSize")
	assert.NotPanics(t, func() { c.rpcRequestsPerRPC.Record(ctx, 0) }, "rpcRequestsPerRPC")
	assert.NotPanics(t, func() { c.rpcResponsesPerRPC.Record(ctx, 0) }, "rpcResponsesPerRPC")
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
