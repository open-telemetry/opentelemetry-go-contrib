// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"fmt"
	__atel_context "context"
	__atel_otel "go.opentelemetry.io/otel"
)

func goroutines(__atel_tracing_ctx __atel_context.Context,) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("goroutines").Start(__atel_tracing_ctx, "goroutines")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()
	messages := make(chan string)

	go func() {
		__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("anonymous").Start(__atel_child_tracing_ctx, "anonymous")
		_ = __atel_child_tracing_ctx
		defer __atel_span.End()
		messages <- "ping"
	}()

	msg := <-messages
	fmt.Println(msg)

}
