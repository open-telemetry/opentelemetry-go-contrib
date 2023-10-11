// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import go.opentelemetry.io/contrib/config

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func initTracerProvider(cfg configOptions) trace.TracerProvider {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return trace.NewNoopTracerProvider()
	}
	return sdktrace.NewTracerProvider()
}
