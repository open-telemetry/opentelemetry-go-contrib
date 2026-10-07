// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spaneventbridge_test

import (
	"context"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/contrib/processors/spaneventbridge"
)

func Example() {
	tp := sdktrace.NewTracerProvider()
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(spaneventbridge.NewLogProcessor()),
	)

	ctx, span := tp.Tracer("Example").Start(context.Background(), "operation")
	defer span.End()

	var record log.Record
	record.SetEventName("cache.miss")
	record.AddAttributes(log.String("cache.key", "user:42"))

	lp.Logger("Example").Emit(ctx, record)
}
