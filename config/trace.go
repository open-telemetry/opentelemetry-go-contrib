// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func initTracerProvider(cfg configOptions) trace.TracerProvider {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return noop.NewTracerProvider()
	}
	return sdktrace.NewTracerProvider()
}
