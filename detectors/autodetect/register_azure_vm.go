// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_azure_vm

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
)

func init() {
	registry[IDAzureVM] = func() resource.Detector { return azurevm.New() }
}
