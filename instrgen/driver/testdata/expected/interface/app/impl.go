// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package app

import (
	"fmt"
	__atel_context "context"
	__atel_otel "go.opentelemetry.io/otel"
)

type BasicSerializer struct {
}

func (b BasicSerializer) Serialize(__atel_tracing_ctx __atel_context.Context,) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("Serialize").Start(__atel_tracing_ctx, "Serialize")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()

	fmt.Println("Serialize")
}
