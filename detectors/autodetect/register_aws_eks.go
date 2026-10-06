// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_eks

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/aws/eks"
)

func init() {
	registry[IDAWSEKS] = eks.NewResourceDetector
}
