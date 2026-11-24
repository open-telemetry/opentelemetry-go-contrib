// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_ec2

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"

	ec2detector "go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
)

func ec2Detector() resource.Detector {
	return ec2detector.NewResourceDetector()
}
