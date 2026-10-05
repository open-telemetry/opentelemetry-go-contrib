// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_gcp

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/gcp"
)

func init() {
	registry[IDGCP] = gcp.NewDetector
}
