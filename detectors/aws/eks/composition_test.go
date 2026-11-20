// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

func testDetector() resource.Detector {
	return &resourceDetector{utils: &arnTestUtils{clusterName: testClusterName}}
}

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK.
func TestComposition_MergeWithDefault(t *testing.T) {
	detected, err := testDetector().Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	res, err := resource.New(t.Context(),
		resource.WithDetectors(testDetector()),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
