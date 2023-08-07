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

package autoexport

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

var stdoutFactory = func(ctx context.Context) (trace.SpanExporter, error) {
	exp, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}
	return exp, nil
}

func TestCanStoreExporterFactory(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		require.NoError(t, r.store("first", stdoutFactory))
	})
}

func TestLoadOfUnknownExporterReturnsError(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		exp, err := r.load(context.Background(), "non-existent")
		assert.Equal(t, err, errUnknownExporter, "empty registry should hold nothing")
		assert.Nil(t, exp, "non-nil exporter returned")
	})
}

func TestRegistryIsConcurrentSafe(t *testing.T) {
	const exporterName = "stdout"

	r := newRegistry()
	assert.NotPanics(t, func() {
		require.NoError(t, r.store(exporterName, stdoutFactory))
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(exporterName, stdoutFactory), errDuplicateRegistration)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NotPanics(t, func() {
			exp, err := r.load(context.Background(), exporterName)
			assert.NoError(t, err, "missing exporter in registry")
			assert.IsType(t, &stdouttrace.Exporter{}, exp)
		})
	}()

	wg.Wait()
}

func TestSubsequentCallsToGetExporterReturnsNewInstances(t *testing.T) {
	const exporterType = "otlp"
	exp1, err := spanExporter(context.Background(), exporterType)
	assert.NoError(t, err)
	assertOTLPHTTPExporter(t, exp1)

	exp2, err := spanExporter(context.Background(), exporterType)
	assert.NoError(t, err)
	assertOTLPHTTPExporter(t, exp2)

	assert.NotSame(t, exp1, exp2)
}

func TestDefaultOTLPExporterFactoriesAreAutomaticallyRegistered(t *testing.T) {
	exp1, err := spanExporter(context.Background(), "")
	assert.Nil(t, err)
	assertOTLPHTTPExporter(t, exp1)

	exp2, err := spanExporter(context.Background(), "otlp")
	assert.Nil(t, err)
	assertOTLPHTTPExporter(t, exp2)
}

func TestEnvRegistryCanRegisterExporterFactory(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	exp, err := envRegistry.load(context.Background(), exporterName)
	assert.Nil(t, err, "missing exporter in envRegistry")
	assert.IsType(t, &stdouttrace.Exporter{}, exp)
}

func TestEnvRegistryPanicsOnDuplicateRegisterCalls(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	errString := fmt.Sprintf("%s: %q", errDuplicateRegistration, exporterName)
	assert.PanicsWithError(t, errString, func() {
		RegisterSpanExporter(exporterName, stdoutFactory)
	})
}
