// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package host // import "go.opentelemetry.io/contrib/instrumentation/host"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// psiMetrics is a no-op type for non-Linux platforms
type psiMetrics struct{}

// registerPSI returns nil on non-Linux platforms (PSI is Linux-specific)
func (h *host) registerPSI() (*psiMetrics, error) {
	return nil, nil
}

// observePSI is a no-op on non-Linux platforms
func (pm *psiMetrics) observePSI(ctx context.Context, o metric.Observer) error {
	return nil
}
