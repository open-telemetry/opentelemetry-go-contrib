// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
)

func testDetector(t *testing.T) resource.Detector {
	t.Helper()
	t.Setenv("GCE_METADATA_HOST", "169.254.169.254")
	return &detector{detector: &fakeGCPDetector{
		projectID:           "my-project",
		cloudPlatform:       gcp.GCE,
		gceHostID:           "1472385723456792345",
		gceHostName:         "my-gke-node-1234",
		gceHostType:         "n1-standard1",
		gceAvailabilityZone: "us-central1-c",
		gceRegion:           "us-central1",
	}}
}

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK. [resource.Merge] reports [resource.ErrSchemaURLConflict]
// and drops the schema URL when the two disagree, so this fails as soon as the
// semconv version here and the one behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	detected, err := testDetector(t).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	d := testDetector(t)

	res, err := resource.New(t.Context(),
		resource.WithDetectors(d),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
