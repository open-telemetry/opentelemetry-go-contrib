// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_hetzner

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/hetzner"
)

func init() {
	registry[IDHetzner] = func() resource.Detector { return hetzner.NewResourceDetector() }
}
