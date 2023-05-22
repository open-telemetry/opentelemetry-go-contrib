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

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
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

func Test_can_store_exporter_factory(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		require.NoError(t, r.store("first", stdoutFactory))
	})
}

func Test_load_of_unknown_exporter_returns_error(t *testing.T) {
	r := newRegistry()
	assert.NotPanics(t, func() {
		exp, err := r.load("non-existent")
		assert.Equal(t, err, errUnknownExpoter, "empty registry should hold nothing")
		assert.Nil(t, exp, "non-nil exporter returned")
	})
}

func Test_registry_is_concurrent_safe(t *testing.T) {
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
			assert.IsType(t, &stdouttrace.Exporter{}, exp)
		})
	}()
}

func Test_subsequent_calls_to_get_exporter_returns_new_instances(t *testing.T) {
	const exporterType = "otlp"
	exp1, err := SpanExporter(exporterType)
	assert.Nil(t, err)
	assert.IsType(t, &otlptrace.Exporter{}, exp1)

	exp2, err := SpanExporter(exporterType)
	assert.Nil(t, err)
	assert.IsType(t, &otlptrace.Exporter{}, exp2)
	assert.NotSame(t, exp1, exp2)
}

func Test_default_otlp_exporter_factories_automatically_registered(t *testing.T) {
	exp1, err := SpanExporter("")
	assert.Nil(t, err)
	assert.IsType(t, &otlptrace.Exporter{}, exp1)

	exp2, err := SpanExporter("otlp")
	assert.Nil(t, err)
	assert.IsType(t, &otlptrace.Exporter{}, exp2)
}

func Test_env_registry_can_register_exporter_factory(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	exp, err := envRegistry.load(exporterName)
	assert.Nil(t, err, "missing exporter in envRegistry")
	assert.IsType(t, &stdouttrace.Exporter{}, exp)
}

func Test_env_registry_panics_on_duplicate_register_calls(t *testing.T) {
	const exporterName = "custom"
	RegisterSpanExporter(exporterName, stdoutFactory)
	t.Cleanup(func() { envRegistry.drop(exporterName) })

	errString := fmt.Sprintf("%s: %q", errDuplicateRegistration, exporterName)
	assert.PanicsWithError(t, errString, func() {
		RegisterSpanExporter(exporterName, stdoutFactory)
	})
}
