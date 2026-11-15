// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
)

func TestWithTextMapPropagator(t *testing.T) {
	cfg := config{}
	propagator := otel.GetTextMapPropagator()

	option := WithTextMapPropagator(propagator)
	option.apply(&cfg)

	assert.Equal(t, cfg.TextMapPropagator, propagator)
}

func TestWithMeterProvider(t *testing.T) {
	cfg := config{}
	provider := otel.GetMeterProvider()

	WithMeterProvider(provider).apply(&cfg)
	assert.Equal(t, provider, cfg.MeterProvider)
}

func TestWithMetricRecorder(t *testing.T) {
	var called []string
	recorder := func(name string) MetricRecorder {
		return func(context.Context, middleware.InitializeInput, middleware.InitializeOutput, error) {
			called = append(called, name)
		}
	}

	cfg := config{}
	WithMetricRecorder(recorder("a"), nil).apply(&cfg)
	WithMetricRecorder(recorder("b")).apply(&cfg)

	for _, r := range cfg.MetricRecorders {
		r(t.Context(), middleware.InitializeInput{}, middleware.InitializeOutput{}, nil)
	}
	assert.Equal(t, []string{"a", "b"}, called)
}
