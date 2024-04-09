// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	__atel_context "context"
	"fmt"

	__atel_otel "go.opentelemetry.io/otel"
)

func foo(__atel_tracing_ctx __atel_context.Context) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("foo").Start(__atel_tracing_ctx, "foo")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()
	fmt.Println("foo")
}

func FibonacciHelper(__atel_tracing_ctx __atel_context.Context, n uint) (uint64, error) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("FibonacciHelper").Start(__atel_tracing_ctx, "FibonacciHelper")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()
	func() {
		__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("anonymous").Start(__atel_child_tracing_ctx, "anonymous")
		_ = __atel_child_tracing_ctx
		defer __atel_span.End()
		foo(__atel_child_tracing_ctx)
	}()
	return Fibonacci(__atel_child_tracing_ctx, n)
}

func Fibonacci(__atel_tracing_ctx __atel_context.Context, n uint) (uint64, error) {
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("Fibonacci").Start(__atel_tracing_ctx, "Fibonacci")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()
	if n <= 1 {
		return uint64(n), nil
	}

	if n > 93 {
		return 0, fmt.Errorf("unsupported fibonacci number %d: too large", n)
	}

	var n2, n1 uint64 = 0, 1
	for i := uint(2); i < n; i++ {
		n2, n1 = n1, n1+n2
	}

	return n2 + n1, nil
}
