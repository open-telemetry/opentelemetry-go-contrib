// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectRenderEnv(t *testing.T) {
	t.Setenv("RENDER_SERVICE_ID", "srv-abc123")
	t.Setenv("RENDER_SERVICE_NAME", "my-service")
	t.Setenv("RENDER_GIT_COMMIT", "abcdef1")

	d := NewResourceDetector()
	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, res.Attributes())
}

func TestDetectRenderEnvMissing(t *testing.T) {
	d := NewResourceDetector()
	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Empty(t, res.Attributes())
}
