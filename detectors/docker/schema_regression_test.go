// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

// TestResourceMergeWithSDKHostDetector is a regression test for a schema URL
// mismatch between this package and go.opentelemetry.io/otel/sdk's own
// built-in detectors.
func TestResourceMergeWithSDKDetector(t *testing.T) {
	detector := newMockDetector(&mockProvider{
		info:          hostInfo{Name: "docker-host", OSType: "linux"},
		containerInfo: containerInfo{Name: "my-container", ImageID: "sha256:deadbeef"},
	})
	dockerRes, err := detector.Detect(t.Context())
	require.NoError(t, err)

	// Host detector has no env dependency and always succeeds
	hostRes, err := resource.New(t.Context(), resource.WithHost())
	require.NoError(t, err)

	_, err = resource.Merge(dockerRes, hostRes)
	require.NoError(t, err, "detector's semconv must match the SDK's built-in detector; see the go.opentelemetry.io/otel/sdk version pinned in go.mod")
}
