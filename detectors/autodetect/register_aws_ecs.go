// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_aws_ecs

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/aws/ecs"
)

func init() {
	registry[IDAWSECS] = ecs.NewResourceDetector
}
