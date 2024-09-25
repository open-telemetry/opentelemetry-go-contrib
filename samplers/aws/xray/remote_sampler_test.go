// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRemoteSamplerDescription assert remote sampling description.
func TestRemoteSamplerDescription(t *testing.T) {
	rs := &remoteSampler{}

	s := rs.Description()
	assert.Equal(t, "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}", s)
}
