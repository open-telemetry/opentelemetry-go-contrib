// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_ecs

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"

	ecsdetector "go.opentelemetry.io/contrib/detectors/aws/ecs"
)

func ecsDetector() resource.Detector {
	return ecsdetector.NewResourceDetector()
}
