// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_azure_containerapps

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
)

func init() {
	registry[IDAzureContainerApps] = func() resource.Detector { return azurecontainerapps.NewResourceDetector() }
}
