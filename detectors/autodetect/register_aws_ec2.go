// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_ec2

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
)

func init() {
	registry[IDAWSEC2] = ec2.NewResourceDetector
}
