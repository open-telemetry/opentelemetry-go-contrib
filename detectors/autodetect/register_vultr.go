// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_vultr

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/vultr"
)

func init() {
	registry[IDVultr] = func() resource.Detector { return vultr.NewResourceDetector() }
}
