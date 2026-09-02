// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
)

// setLambdaEnv puts the process in a Lambda-like environment so the detector
// returns a populated resource rather than an empty one.
func setLambdaEnv(t *testing.T) {
	t.Helper()
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogGroupNameEnvVar, "/aws/lambda/testFunction")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")
}

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK. [resource.Merge] reports [resource.ErrSchemaURLConflict]
// and drops the schema URL when the two disagree, so this fails as soon as the
// semconv version here and the one behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	setLambdaEnv(t)

	detected, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	setLambdaEnv(t)

	res, err := resource.New(t.Context(),
		resource.WithDetectors(NewResourceDetector()),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
