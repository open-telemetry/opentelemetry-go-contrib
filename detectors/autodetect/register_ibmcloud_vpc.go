// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_ibmcloud_vpc

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"
)

func init() {
	registry[IDIBMCloudVPC] = func() resource.Detector { return vpc.NewResourceDetector() }
}
