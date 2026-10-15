// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nova

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
)

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK. [resource.Merge] reports [resource.ErrSchemaURLConflict]
// and drops the schema URL when the two disagree, so this fails as soon as the
// semconv version here and the one behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	detected, err := newTestDetector(srv.url).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	require.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	srv := newFakeServer(t, fullMetadata(), "m1.medium")

	res, err := resource.New(t.Context(),
		resource.WithDetectors(newTestDetector(srv.url)),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	require.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
