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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

var stdoutFactory = func() (trace.SpanExporter, error) {
	exp, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}
	return exp, nil
}

func TestRegistryEmptyStore(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		require.NoError(t, r.store("first", stdoutFactory))
	})
}

func TestRegistryEmptyLoad(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		exp, err := r.load("non-existent")
		assert.Equal(t, err, errUnknownExpoter, "empty registry should hold nothing")
		assert.Nil(t, exp, "non-nil exporter returned")
	})
}

func TestRegistryConcurrentSafe(t *testing.T) {
	const exporterName = "stdout"

	r := newRegistry()
	assert.NotPanics(t, func() {
		require.NoError(t, r.store(exporterName, stdoutFactory))
	})

	go func() {
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(exporterName, stdoutFactory), errDuplicateRegistration)
		})
	}()

	go func() {
		assert.NotPanics(t, func() {
			exp, err := r.load(exporterName)
			assert.Nil(t, err, "missing exporter in registry")
			_, ok := exp.(*stdouttrace.Exporter)
			if !ok {
				assert.Fail(t, "wrong exporter retuned")
			}
		})
	}()
}

func TestRegisterSpanExporter(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	exp, err := envRegistry.load(exporterName)
	assert.Nil(t, err, "missing exporter in envRegistry")
	_, ok := exp.(*stdouttrace.Exporter)
	if !ok {
		assert.Fail(t, "wrong exporter stored")
	}
}

func TestDuplicateRegisterSpanExporterPanics(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	errString := fmt.Sprintf("%s: %q", errDuplicateRegistration, exporterName)
	assert.PanicsWithError(t, errString, func() {
		RegisterSpanExporter(exporterName, stdoutFactory)
	})
}

func TestRetrievingSameKeyReturnsDifferentExporterInstance(t *testing.T) {
	const exporterType = "otlp"
	exp1, err := SpanExporter(exporterType)
	assert.Nil(t, err)

	exp2, err := SpanExporter(exporterType)
	assert.Nil(t, err)
	assert.NotEqual(t, exp1, exp2)
}

func TestOTLPExporterIsAutomaticallyRegistered(t *testing.T) {
	exp1, err := SpanExporter("")
	assert.Nil(t, err)
	assert.NotNil(t, exp1)

	exp2, err := SpanExporter("otlp")
	assert.Nil(t, err)
	assert.NotNil(t, exp2)
}
