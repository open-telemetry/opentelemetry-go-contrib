// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_openshift

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/openshift"
)

func init() {
	registry[IDOpenShift] = func() resource.Detector { return openshift.NewResourceDetector() }
}
