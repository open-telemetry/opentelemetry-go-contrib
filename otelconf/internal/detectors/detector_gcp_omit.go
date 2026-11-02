// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build omit_detector_gcp

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"
)

func gcpDetector() resource.Detector {
	return nil
}
