// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoprop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
)

var noop = propagation.NewCompositeTextMapPropagator()

func TestRegistryEmptyStore(t *testing.T) {
	r := registry{}
	assert.NotPanics(t, func() {
		require.NoError(t, r.store("first", noop))
	})
}

func TestRegistryEmptyLoad(t *testing.T) {
	r := registry{}
	assert.NotPanics(t, func() {
		v, ok := r.load("non-existent")
		assert.False(t, ok, "empty registry should hold nothing")
		assert.Nil(t, v, "non-nil propagator returned")
	})
}

func TestRegistryConcurrentSafe(t *testing.T) {
	const propName = "prop"

	r := registry{}
	assert.NotPanics(t, func() {
		require.NoError(t, r.store(propName, noop))
	})

	var wg sync.WaitGroup

	wg.Go(func() {
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(propName, noop), errDupReg)
		})
	})

	wg.Go(func() {
		assert.NotPanics(t, func() {
			v, ok := r.load(propName)
			assert.True(t, ok, "missing propagator in registry")
			assert.Equal(t, noop, v, "wrong propagator returned")
		})
	})

	wg.Wait()
}

func TestRegisterTextMapPropagator(t *testing.T) {
	const propName = "custom"
	RegisterTextMapPropagator(propName, noop)
	t.Cleanup(func() { propagators.drop(propName) })

	v, ok := propagators.load(propName)
	assert.True(t, ok, "missing propagator in envRegistry")
	assert.Equal(t, noop, v, "wrong propagator stored")
}

func TestDuplicateRegisterTextMapPropagatorPanics(t *testing.T) {
	const propName = "custom"
	RegisterTextMapPropagator(propName, noop)
	t.Cleanup(func() { propagators.drop(propName) })

	errString := fmt.Sprintf("%s: %q", errDupReg, propName)
	assert.PanicsWithError(t, errString, func() {
		RegisterTextMapPropagator(propName, noop)
	})
}

func TestTextMapPropagatorEmptyNoError(t *testing.T) {
	p, err := TextMapPropagator()
	require.NoError(t, err)
	assert.Equal(t, noopPropagator, p, "empty input should return the no-op propagator")
}

func TestTextMapPropagatorAllUnknown(t *testing.T) {
	// Names that do not match any registered propagator must return a nil
	// propagator along with the error. NewTextMapPropagator relies on the nil
	// to fall back to its default instead of disabling propagation.
	p, err := TextMapPropagator("invalid")
	require.ErrorIs(t, err, errUnknownPropagator)
	assert.Nil(t, p, "all-unknown input should return a nil propagator")
}

type recordingPropagator struct {
	name  string
	calls *[]string
}

func (p recordingPropagator) Inject(context.Context, propagation.TextMapCarrier) {
	*p.calls = append(*p.calls, "inject "+p.name)
}

func (p recordingPropagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	*p.calls = append(*p.calls, "extract "+p.name)
	return ctx
}

func (recordingPropagator) Fields() []string { return nil }

func TestTextMapPropagatorDeduplicates(t *testing.T) {
	for _, fromEnv := range []bool{false, true} {
		t.Run(fmt.Sprintf("fromEnv=%t", fromEnv), func(t *testing.T) {
			var calls []string
			first, second := t.Name()+"-first", t.Name()+"-second"
			RegisterTextMapPropagator(first, recordingPropagator{"first", &calls})
			RegisterTextMapPropagator(second, recordingPropagator{"second", &calls})
			t.Cleanup(func() {
				propagators.drop(first)
				propagators.drop(second)
			})

			names := []string{first, second, first, second}
			var p propagation.TextMapPropagator
			if fromEnv {
				t.Setenv(otelPropagatorsEnvKey, strings.Join(names, ","))
				p = NewTextMapPropagator()
			} else {
				var err error
				p, err = TextMapPropagator(names...)
				require.NoError(t, err)
			}
			p.Inject(t.Context(), propagation.MapCarrier{})
			p.Extract(t.Context(), propagation.MapCarrier{})
			assert.Equal(t, []string{"inject first", "inject second", "extract first", "extract second"}, calls)
		})
	}
}
