// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_azure_vm

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"

	azurevmdetector "go.opentelemetry.io/contrib/detectors/azure/azurevm"
)

func azurevmDetector() resource.Detector {
	return azurevmdetector.NewResourceDetector()
}
