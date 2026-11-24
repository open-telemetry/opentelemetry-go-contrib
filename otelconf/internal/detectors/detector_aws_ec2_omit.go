// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build omit_detector_aws_ec2

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"
)

func ec2Detector() resource.Detector {
	return nil
}
