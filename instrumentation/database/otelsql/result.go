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
	_ driver.Result = otelResult{}
)

// otelResult implements driver.Result
type otelResult struct {
	parent  driver.Result
	ctx     context.Context
	options wrapper
}

func (r otelResult) LastInsertId() (id int64, err error) {
	if r.options.LastInsertID {
		attrs := append([]attribute.KeyValue(nil), r.options.DefaultAttributes...)
		ctx := r.ctx
		onDeferWithErr := recordCallStats("go.sql.result.last_insert_id", r.options.InstanceName)
		defer func() {
			// Invoking this function in a defer so that we can capture
			// the value of err as set on function exit.
			onDeferWithErr(ctx, err, attrs...)
		}()

		parentSpan := trace.SpanFromContext(ctx)
		if !r.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			// we already tested driver
			return r.parent.LastInsertId()
		}

		ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, r.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
		span.SetAttributes(attrs...)
	}

	id, err = r.parent.LastInsertId()
	return
}

func (r otelResult) RowsAffected() (cnt int64, err error) {
	if r.options.RowsAffected {
		attrs := append([]attribute.KeyValue(nil), r.options.DefaultAttributes...)
		ctx := r.ctx
		onDeferWithErr := recordCallStats("go.sql.result.rows_affected", r.options.InstanceName)
		defer func() {
			// Invoking this function in a defer so that we can capture
			// the value of err as set on function exit.
			onDeferWithErr(ctx, err, attrs...)
		}()

		parentSpan := trace.SpanFromContext(ctx)
		if !r.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			// we already tested driver
			return r.parent.LastInsertId()
		}

		ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, r.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
		span.SetAttributes(attrs...)
	}

	cnt, err = r.parent.RowsAffected()
	return
}
