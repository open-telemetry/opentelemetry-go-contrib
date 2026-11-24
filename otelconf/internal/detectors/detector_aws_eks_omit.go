// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build omit_detector_aws_eks

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"
)

func eksDetector() resource.Detector {
	return nil
}
