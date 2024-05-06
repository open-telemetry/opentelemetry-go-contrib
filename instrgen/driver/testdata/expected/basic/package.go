// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"os"
	__atel_context "context"
	__atel_otel "go.opentelemetry.io/otel"
)

func Close() error {
	return nil
}

func pack(__atel_tracing_ctx __atel_context.Context,) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("pack").Start(__atel_tracing_ctx, "pack")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()
	f, e := os.Create("temp")
	defer f.Close()
	if e != nil {

	}
}
