// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoprop

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type handler struct {
	err error
}

func (h *handler) Handle(err error) { h.err = err }

func TestNewTextMapPropagatorInvalidEnvVal(t *testing.T) {
	h := &handler{}
	otel.SetErrorHandler(h)

	const name = "invalid-name"
	t.Setenv(otelPropagatorsEnvKey, name)
	_ = NewTextMapPropagator()
	assert.ErrorIs(t, h.err, errUnknownPropagator)
}

func TestNewTextMapPropagatorDefault(t *testing.T) {
	expect := []string{"traceparent", "tracestate", "baggage"}
	assert.ElementsMatch(t, expect, NewTextMapPropagator().Fields())
}

func TestEmptyElementIsIgnored(t *testing.T) {
	in := []string{"", "tracecontext"}
	p, err := TextMapPropagator(in...)
	require.NoError(t, err)
	assert.Equal(t, propagation.TraceContext{}, p)
}

func TestEmptyElementSet(t *testing.T) {
	in := []string{}
	p, err := TextMapPropagator(in...)
	require.NoError(t, err)
	assert.Equal(t, propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}), p)
}

type ptrNoop struct{}

func (*ptrNoop) Inject(context.Context, propagation.TextMapCarrier) {}

func (*ptrNoop) Extract(context.Context, propagation.TextMapCarrier) context.Context {
	return context.Background()
}

func (*ptrNoop) Fields() []string {
	return nil
}

func TestNewTextMapPropagatorSingleNoOverhead(t *testing.T) {
	p := &ptrNoop{}
	assert.Same(t, p, NewTextMapPropagator(p))
}

func TestNewTextMapPropagatorMultiEnvNone(t *testing.T) {
	t.Setenv(otelPropagatorsEnvKey, "b3,none,tracecontext")
	assert.Equal(t, noop, NewTextMapPropagator())
}
