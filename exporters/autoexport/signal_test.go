// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
