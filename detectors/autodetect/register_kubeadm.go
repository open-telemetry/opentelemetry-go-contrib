// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_kubeadm

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/kubeadm"
)

func init() {
	registry[IDKubeadm] = func() resource.Detector { return kubeadm.NewResourceDetector() }
}
