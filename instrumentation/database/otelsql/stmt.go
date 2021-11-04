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
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Stmt             = otelStmt{}
	_ driver.StmtExecContext  = otelStmt{}
	_ driver.StmtQueryContext = otelStmt{}
	_ driver.ColumnConverter  = otelStmt{}
)

// otelStmt implements driver.Stmt
type otelStmt struct {
	parent  driver.Stmt
	query   string
	options wrapper
}

func (s otelStmt) ColumnConverter(idx int) driver.ValueConverter {
	if converter, ok := s.parent.(driver.ColumnConverter); ok {
		return converter.ColumnConverter(idx)
	}

	return driver.DefaultParameterConverter
}

func (s otelStmt) Exec(args []driver.Value) (res driver.Result, err error) {
	ctx := context.Background()
	attrs := append([]attribute.KeyValue(nil), s.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.stmt.exec", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if !s.options.AllowRoot {
		return s.parent.Exec(args)
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, s.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()

	attrs = append(
		attrs,
		attrDeprecated,
		attribute.String("sql.deprecated", "driver does not support StmtExecContext"),
	)

	if s.options.Query {
		attrs = append(attrs, semconv.DBStatementKey.String(s.query))
		if s.options.QueryParams {
			attrs = append(attrs, paramsAttr(args)...)
		}
	}

	res, err = s.parent.Exec(args)
	if err != nil {
		return nil, err
	}

	res, err = otelResult{parent: res, ctx: ctx, options: s.options}, nil
	return
}

func (s otelStmt) Close() error {
	return s.parent.Close()
}

func (s otelStmt) NumInput() int {
	return s.parent.NumInput()
}

func (s otelStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	ctx := context.Background()
	attrs := append([]attribute.KeyValue(nil), s.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.stmt.query", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if !s.options.AllowRoot {
		return s.parent.Query(args)
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:query", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, s.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()

	attrs = append(
		attrs,
		attrDeprecated,
		attribute.String("sql.deprecated", "driver does not support StmtQueryContext"),
	)

	if s.options.Query {
		attrs = append(attrs, semconv.DBStatementKey.String(s.query))
		if s.options.QueryParams {
			attrs = append(attrs, paramsAttr(args)...)
		}
	}

	rows, err = s.parent.Query(args)
	if err != nil {
		return nil, err
	}
	rows, err = wrapRows(ctx, rows, s.options), nil
	return
}

func (s otelStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	attrs := append([]attribute.KeyValue(nil), s.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.stmt.exec", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if !s.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
		// we already tested driver to implement StmtExecContext
		return s.parent.(driver.StmtExecContext).ExecContext(ctx, args)
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, s.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()
	if s.options.Query {
		attrs = append(attrs, semconv.DBStatementKey.String(s.query))
		if s.options.QueryParams {
			attrs = append(attrs, namedParamsAttr(args)...)
		}
	}

	// we already tested driver to implement StmtExecContext
	execContext := s.parent.(driver.StmtExecContext)
	res, err = execContext.ExecContext(ctx, args)
	if err != nil {
		return nil, err
	}
	res, err = otelResult{parent: res, ctx: ctx, options: s.options}, nil
	return
}

func (s otelStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	var attrs []attribute.KeyValue
	onDeferWithErr := recordCallStats("go.sql.stmt.query", s.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if !s.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
		// we already tested driver to implement StmtExecContext
		return s.parent.(driver.StmtQueryContext).QueryContext(ctx, args)
	}

	ctx, span := otel.Tracer("").Start(ctx, "sql:query", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, s.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()

	if s.options.Query {
		attrs = append(attrs, semconv.DBStatementKey.String(s.query))
		if s.options.QueryParams {
			attrs = append(attrs, namedParamsAttr(args)...)
		}
	}

	// we already tested driver to implement StmtQueryContext
	queryContext := s.parent.(driver.StmtQueryContext)
	rows, err = queryContext.QueryContext(ctx, args)
	if err != nil {
		return nil, err
	}
	rows, err = wrapRows(ctx, rows, s.options), nil
	return
}
