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
	"fmt"
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

	go func() {
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(propName, noop), errDupReg)
		})
	}()

	go func() {
		assert.NotPanics(t, func() {
			v, ok := r.load(propName)
			assert.True(t, ok, "missing propagator in registry")
			assert.Equal(t, noop, v, "wrong propagator retuned")
		})
	}()
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
