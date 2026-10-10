// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_ecs

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSDetector(t *testing.T) {
	assert.NotNil(t, ecsDetector())
}
