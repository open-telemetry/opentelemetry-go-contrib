// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_gcp

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/gcp"
)

func init() {
	registry[IDGCP] = gcp.NewDetector
}
