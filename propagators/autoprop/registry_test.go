// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoprop

import (
	"fmt"
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(propName, noop), errDupReg)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NotPanics(t, func() {
			v, ok := r.load(propName)
			assert.True(t, ok, "missing propagator in registry")
			assert.Equal(t, noop, v, "wrong propagator returned")
		})
	}()

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
