// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

import (
	"context"
	"database/sql/driver"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Tx = otelTx{}
)

// otelTx implements driver.Tx
type otelTx struct {
	parent  driver.Tx
	ctx     context.Context
	options wrapper
}

func (t otelTx) Commit() (err error) {
	ctx := t.ctx
	attrs := append([]attribute.KeyValue(nil), t.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.tx.commit", t.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if !t.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
		// we already tested driver
		return t.parent.Commit()
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:commit", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, t.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()
	span.SetAttributes(attrs...)

	err = t.parent.Commit()
	return
}

func (t otelTx) Rollback() (err error) {
	ctx := t.ctx
	var attrs []attribute.KeyValue
	onDeferWithErr := recordCallStats("go.sql.tx.rollback", t.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if !t.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
		// we already tested driver
		return t.parent.Commit()
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:rollback", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, t.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()
	span.SetAttributes(attrs...)
	err = t.parent.Rollback()
	return
}
