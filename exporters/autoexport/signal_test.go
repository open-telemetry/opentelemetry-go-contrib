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

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOTLPExporterReturnedWhenNoEnvOrFallbackExporterConfigured(t *testing.T) {
	ts := newSignal[*testType]("TEST_TYPE_KEY")
	assert.NoError(t, ts.registry.store("otlp", factory("test-otlp-exporter")))
	exp, err := ts.create(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, exp.string, "test-otlp-exporter")
}

func TestFallbackExporterReturnedWhenNoEnvExporterConfigured(t *testing.T) {
	ts := newSignal[*testType]("TEST_TYPE_KEY")
	fallback := testType{"test-fallback-exporter"}
	exp, err := ts.create(context.Background(), withFallback(&fallback))
	assert.NoError(t, err)
	assert.Same(t, &fallback, exp)
}

func TestEnvExporterIsPreferredOverFallbackExporter(t *testing.T) {
	envVariable := "TEST_TYPE_KEY"
	ts := newSignal[*testType](envVariable)

	expName := "test-env-exporter-name"
	t.Setenv(envVariable, expName)
	fallback := testType{"test-fallback-exporter"}
	assert.NoError(t, ts.registry.store(expName, factory("test-env-exporter")))

	exp, err := ts.create(context.Background(), withFallback(&fallback))
	assert.NoError(t, err)
	assert.Equal(t, exp.string, "test-env-exporter")
}
