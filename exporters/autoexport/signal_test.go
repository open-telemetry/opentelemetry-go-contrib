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
	"errors"
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
	exp, err := ts.create(context.Background(), withFallbackFactory(factory("test-fallback-exporter")))
	assert.NoError(t, err)
	assert.Equal(t, exp.string, "test-fallback-exporter")
}

func TestFallbackExporterFactoryErrorReturnedWhenNoEnvExporterConfiguredAndFallbackFactoryReturnsAnError(t *testing.T) {
	ts := newSignal[*testType]("TEST_TYPE_KEY")

	expectedErr := errors.New("error expected to return")
	errFactory := func(ctx context.Context) (*testType, error) {
		return nil, expectedErr
	}
	exp, err := ts.create(context.Background(), withFallbackFactory(errFactory))
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, exp)
}

func TestEnvExporterIsPreferredOverFallbackExporter(t *testing.T) {
	envVariable := "TEST_TYPE_KEY"
	ts := newSignal[*testType](envVariable)

	expName := "test-env-exporter-name"
	t.Setenv(envVariable, expName)
	assert.NoError(t, ts.registry.store(expName, factory("test-env-exporter")))

	exp, err := ts.create(context.Background(), withFallbackFactory(factory("test-fallback-exporter")))
	assert.NoError(t, err)
	assert.Equal(t, exp.string, "test-env-exporter")
}
