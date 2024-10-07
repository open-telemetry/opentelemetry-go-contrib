// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testType struct{ string }

func factory(val string) func(ctx context.Context) (*testType, error) {
	return func(ctx context.Context) (*testType, error) { return &testType{val}, nil }
}

func newTestRegistry() registry[*testType] {
	return registry[*testType]{
		names: make(map[string]func(context.Context) (*testType, error)),
	}
}

func TestCanStoreExporterFactory(t *testing.T) {
	r := newTestRegistry()
	require.NoError(t, r.store("first", factory("first")))
}

func TestLoadOfUnknownExporterReturnsError(t *testing.T) {
	r := newTestRegistry()
	exp, err := r.load(context.Background(), "non-existent")
	assert.Equal(t, err, errUnknownExporterProducer, "empty registry should hold nothing")
	assert.Nil(t, exp, "non-nil exporter returned")
}

func TestRegistryIsConcurrentSafe(t *testing.T) {
	const exporterName = "stdout"

	r := newTestRegistry()
	require.NoError(t, r.store(exporterName, factory("stdout")))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.ErrorIs(t, r.store(exporterName, factory("stdout")), errDuplicateRegistration)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := r.load(context.Background(), exporterName)
		assert.NoError(t, err, "missing exporter in registry")
	}()

	wg.Wait()
}

func TestSubsequentCallsToGetExporterReturnsNewInstances(t *testing.T) {
	r := newTestRegistry()

	const key = "key"
	assert.NoError(t, r.store(key, factory(key)))

	exp1, err := r.load(context.Background(), key)
	assert.NoError(t, err)

	exp2, err := r.load(context.Background(), key)
	assert.NoError(t, err)

	assert.NotSame(t, exp1, exp2)
}

func TestRegistryErrorsOnDuplicateRegisterCalls(t *testing.T) {
	r := newTestRegistry()

	const exporterName = "custom"
	assert.NoError(t, r.store(exporterName, factory(exporterName)))

	errString := fmt.Sprintf("%s: %q", errDuplicateRegistration, exporterName)
	assert.ErrorContains(t, r.store(exporterName, factory(exporterName)), errString)
}

func TestMust(t *testing.T) {
	assert.Panics(t, func() { must(errors.New("test")) })
	assert.NotPanics(t, func() { must(nil) })
}
