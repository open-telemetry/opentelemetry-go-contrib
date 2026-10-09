// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build omit_detector_gcp

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGCPDetector(t *testing.T) {
	assert.Nil(t, gcpDetector())
}
