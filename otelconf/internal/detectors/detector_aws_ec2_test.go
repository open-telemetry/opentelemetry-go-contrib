// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_ec2

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEC2Detector(t *testing.T) {
	assert.NotNil(t, ec2Detector())
}
