// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_lambda

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAWSLambdaDetector(t *testing.T) {
	assert.NotNil(t, awsLambdaDetector())
}
