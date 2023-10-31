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

package autoprop

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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
