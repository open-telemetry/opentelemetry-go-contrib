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
)

var stdout = &stdouttrace.Exporter{}

func TestRegistryEmptyStore(t *testing.T) {
	r := registry{}
	assert.NotPanics(t, func() {
		require.NoError(t, r.store("first", stdout))
	})
}

func TestRegistryEmptyLoad(t *testing.T) {
	r := registry{}
	assert.NotPanics(t, func() {
		v, ok := r.load("non-existent")
		assert.False(t, ok, "empty registry should hold nothing")
		assert.Nil(t, v, "non-nil exporter returned")
	})
}

func TestRegistryConcurrentSafe(t *testing.T) {
	const exporterName = "stdout"

	r := registry{}
	assert.NotPanics(t, func() {
		require.NoError(t, r.store(exporterName, stdout))
	})

	go func() {
		assert.NotPanics(t, func() {
			require.ErrorIs(t, r.store(exporterName, stdout), errDuplicateRegistration)
		})
	}()

	go func() {
		assert.NotPanics(t, func() {
			v, ok := r.load(exporterName)
			assert.True(t, ok, "missing exporter in registry")
			assert.Equal(t, stdout, v, "wrong exporter retuned")
		})
	}()
}

func TestRegisterSpanExporter(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdout)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	v, ok := envRegistry.load(exporterName)
	assert.True(t, ok, "missing exporter in envRegistry")
	assert.Equal(t, stdout, v, "wrong exporter stored")
}

func TestDuplicateRegisterSpanExporterPanics(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdout)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	errString := fmt.Sprintf("%s: %q", errDuplicateRegistration, exporterName)
	assert.PanicsWithError(t, errString, func() {
		RegisterSpanExporter(exporterName, stdout)
	})
}

func TestRetrievingSameKeyReturnsSameExporterInstance(t *testing.T) {
	const exporterType = "otlp"
	exp1, err := SpanExporter(exporterType)
	assert.Nil(t, err)

	exp2, err := SpanExporter(exporterType)
	assert.Nil(t, err)
	assert.Equal(t, exp1, exp2)
}

func TestOTLPExporterIsAutomaticallyRegistered(t *testing.T) {
	exp1, err := SpanExporter("")
	assert.Nil(t, err)

	exp2, err := SpanExporter("otlp")
	assert.Nil(t, err)

	assert.Equal(t, exp1, exp2)
}
