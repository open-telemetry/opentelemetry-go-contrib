// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"testing"

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
