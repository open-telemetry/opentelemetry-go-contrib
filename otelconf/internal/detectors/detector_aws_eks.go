// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_eks

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"

	eksdetector "go.opentelemetry.io/contrib/detectors/aws/eks"
)

func eksDetector() resource.Detector {
	return eksdetector.NewResourceDetector()
}
