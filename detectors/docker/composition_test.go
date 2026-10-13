// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
)

func testDetector(t *testing.T) *ResourceDetector {
	t.Helper()
	return newMockDetector(&mockProvider{
		info:          hostInfo{Name: "docker-host", OSType: "linux"},
		containerInfo: containerInfo{Name: "my-container", ImageID: "sha256:deadbeef"},
	})
}

// TestComposition_MergeWithDefault guards against schema URL drift between
// this detector and the SDK. [resource.Merge] reports
// [resource.ErrSchemaURLConflict] and drops the schema URL when the two
// disagree, so this fails as soon as the semconv version here and the one
// behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	detected, err := testDetector(t).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with
// go.opentelemetry.io/otel/sdk's own built-in host detector, the specific
// conflict this guards against: see
// https://github.com/open-telemetry/opentelemetry-go-contrib/pull/9001#discussion_r3677186896.
func TestComposition_WithCoreDetectors(t *testing.T) {
	d := testDetector(t)

	res, err := resource.New(t.Context(),
		resource.WithDetectors(d),
		resource.WithHost(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
