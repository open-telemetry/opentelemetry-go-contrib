// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_docker

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/docker"
)

func init() {
	registry[IDDocker] = func() resource.Detector { return docker.NewResourceDetector() }
}
