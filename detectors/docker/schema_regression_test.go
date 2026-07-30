// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// TestResourceMergeWithSDKHostDetector is a regression test for a schema URL
// mismatch between this package and go.opentelemetry.io/otel/sdk's own
// built-in detectors.
func TestResourceMergeWithSDKDetector(t *testing.T) {
	dockerRes := resource.NewWithAttributes(semconv.SchemaURL, semconv.ContainerName("test-container"))

	// Host detector has no env dependency and always succeeds
	hostRes, err := resource.New(t.Context(), resource.WithHost())
	require.NoError(t, err)

	_, err = resource.Merge(dockerRes, hostRes)
	require.NoError(t, err, "detector's semconv must match the SDK's built-in detector; see the go.opentelemetry.io/otel/sdk version pinned in go.mod")
}
