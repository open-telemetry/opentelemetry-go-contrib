// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	__atel_context "context"

	"go.opentelemetry.io/contrib/instrgen/rtlib"
	. "go.opentelemetry.io/contrib/instrgen/testdata/interface/app"
	. "go.opentelemetry.io/contrib/instrgen/testdata/interface/serializer"
	__atel_otel "go.opentelemetry.io/otel"
)

func main() {
	__atel_ts := rtlib.NewTracingState()
	defer rtlib.Shutdown(__atel_ts)
	__atel_otel.SetTracerProvider(__atel_ts.Tp)
	__atel_ctx := __atel_context.Background()
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("main").Start(__atel_ctx, "main")
	_ = __atel_child_tracing_ctx
	defer __atel_span.End()

	rtlib.AutotelEntryPoint()
	bs := BasicSerializer{}
	var s Serializer
	s = bs
	s.Serialize(__atel_child_tracing_ctx)
}
