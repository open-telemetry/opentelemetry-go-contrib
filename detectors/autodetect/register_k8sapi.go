// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_k8sapi

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/k8sapi"
)

func init() {
	registry[IDK8sAPI] = func() resource.Detector { return k8sapi.NewResourceDetector() }
}
