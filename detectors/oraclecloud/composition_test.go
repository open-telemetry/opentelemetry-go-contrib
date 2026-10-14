// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
)

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK. [resource.Merge] reports [resource.ErrSchemaURLConflict]
// and drops the schema URL when the two disagree, so this fails as soon as the
// semconv version here and the one behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		AvailabilityDomain: "AD-1",
	})

	detected, err := newTestDetector(url).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	url := newFakeServer(t, computeMetadata{
		HostID:             "ocid1.instance.oc1..aaaaaaa",
		HostDisplayName:    "my-instance",
		HostType:           "VM.Standard.E4.Flex",
		CanonicalRegionID:  "us-ashburn-1",
		AvailabilityDomain: "AD-1",
	})
	d := newTestDetector(url)

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
