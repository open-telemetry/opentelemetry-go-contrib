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

type conn interface {
	driver.Pinger
	driver.Execer
	driver.ExecerContext
	driver.Queryer
	driver.QueryerContext
	driver.Conn
	driver.ConnPrepareContext
	driver.ConnBeginTx
}

var (
	// Compile time assertions
	_ conn                     = &otelConn{}
	_ driver.NamedValueChecker = &otelConn{}
)

// WrapConn allows an existing driver.Conn to be wrapped by sql.
func WrapConn(c driver.Conn, options ...WrapperOption) driver.Conn {
	var o wrapper
	o.SetDefaults()
	o.ApplyOptions(options...)
	if o.InstanceName == "" {
		o.InstanceName = defaultInstanceName
	} else {
		o.DefaultAttributes = append(o.DefaultAttributes, attribute.String("sql.instance", o.InstanceName))
	}
	return wrapConn(c, o)
}

// otelConn implements driver.Conn
type otelConn struct {
	parent  driver.Conn
	options wrapper
}

func (c otelConn) Ping(ctx context.Context) (err error) {
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.ping", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)

	if c.options.Ping && (c.options.AllowRoot && parentSpan.SpanContext().IsValid()) {
		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:ping", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
	}

	if pinger, ok := c.parent.(driver.Pinger); ok {
		err = pinger.Ping(ctx)
	}
	return
}

func (c otelConn) Exec(query string, args []driver.Value) (res driver.Result, err error) {
	ctx := context.Background()
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.exec", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if exec, ok := c.parent.(driver.Execer); ok {
		if !c.options.AllowRoot {
			return exec.Exec(query, args)
		}

		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()

		attrs = append(
			attrs,
			attrDeprecated,
			attribute.String("sql.deprecated", "driver does not support ExecerContext"),
		)

		if res, err = exec.Exec(query, args); err != nil {
			return nil, err
		}

		return otelResult{parent: res, ctx: ctx, options: c.options}, nil
	}

	return nil, driver.ErrSkip
}

func (c otelConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (res driver.Result, err error) {
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.exec", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if execCtx, ok := c.parent.(driver.ExecerContext); ok {
		parentSpan := trace.SpanFromContext(ctx)
		if !c.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			return execCtx.ExecContext(ctx, query, args)
		}

		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
		if c.options.Query {
			attrs = append(attrs, semconv.DBStatementKey.String(query))
			if c.options.QueryParams {
				attrs = append(attrs, namedParamsAttr(args)...)
			}
		}

		if res, err = execCtx.ExecContext(ctx, query, args); err != nil {
			return nil, err
		}

		return otelResult{parent: res, ctx: ctx, options: c.options}, nil
	}

	return nil, driver.ErrSkip
}

func (c otelConn) Query(query string, args []driver.Value) (rows driver.Rows, err error) {
	ctx := context.Background()
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.query", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if queryer, ok := c.parent.(driver.Queryer); ok {
		if !c.options.AllowRoot {
			return queryer.Query(query, args)
		}

		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()

		attrs = append(
			attrs,
			attrDeprecated,
			attribute.String("sql.deprecated", "driver does not support QueryerContext"),
		)
		if c.options.Query {
			attrs = append(attrs, semconv.DBStatementKey.String(query))
			if c.options.QueryParams {
				attrs = append(attrs, paramsAttr(args)...)
			}
		}

		rows, err = queryer.Query(query, args)
		if err != nil {
			return nil, err
		}

		return wrapRows(ctx, rows, c.options), nil
	}

	return nil, driver.ErrSkip
}

func (c otelConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.query", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if queryerCtx, ok := c.parent.(driver.QueryerContext); ok {
		parentSpan := trace.SpanFromContext(ctx)
		if !c.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			return queryerCtx.QueryContext(ctx, query, args)
		}

		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
		if c.options.Query {
			attrs = append(attrs, semconv.DBStatementKey.String(query))
			if c.options.QueryParams {
				attrs = append(attrs, namedParamsAttr(args)...)
			}
		}

		rows, err = queryerCtx.QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}

		return wrapRows(ctx, rows, c.options), nil
	}

	return nil, driver.ErrSkip
}

func (c otelConn) Prepare(query string) (stmt driver.Stmt, err error) {
	ctx := context.Background()
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.prepare", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	if c.options.AllowRoot {
		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:prepare", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()

		attrs = append(
			attrs,
			attrMissingContext,
		)
		if c.options.Query {
			attrs = append(attrs, semconv.DBStatementKey.String(query))
		}
	}

	stmt, err = c.parent.Prepare(query)
	if err != nil {
		return nil, err
	}

	stmt = wrapStmt(stmt, query, c.options)
	return
}

func (c *otelConn) Close() error {
	return c.parent.Close()
}

func (c *otelConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.TODO(), driver.TxOptions{})
}

func (c *otelConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.prepare", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if c.options.AllowRoot || parentSpan.SpanContext().IsValid() {
		var span trace.Span
		ctx, span = otel.Tracer("").Start(ctx, "sql:prepare", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, c.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
		if c.options.Query {
			attrs = append(attrs, semconv.DBStatementKey.String(query))
		}
	}

	if prepCtx, ok := c.parent.(driver.ConnPrepareContext); ok {
		stmt, err = prepCtx.PrepareContext(ctx, query)
	} else {
		attrs = append(attrs, attrMissingContext)
		stmt, err = c.parent.Prepare(query)
	}

	if err != nil {
		return nil, err
	}

	stmt = wrapStmt(stmt, query, c.options)
	return
}

func (c *otelConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	attrs := append([]attribute.KeyValue(nil), c.options.DefaultAttributes...)
	onDeferWithErr := recordCallStats("go.sql.conn.begin_tx", c.options.InstanceName)
	defer func() {
		// Invoking this function in a defer so that we can capture
		// the value of err as set on function exit.
		onDeferWithErr(ctx, err, attrs...)
	}()

	parentSpan := trace.SpanFromContext(ctx)
	if !c.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
		if connBeginTx, ok := c.parent.(driver.ConnBeginTx); ok {
			return connBeginTx.BeginTx(ctx, opts)
		}
		return c.parent.Begin()
	}
	var span trace.Span
	ctx, span = otel.Tracer("").Start(ctx, "sql:begin_transaction", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		setSpanStatus(span, c.options, err)
		span.SetAttributes(attrs...)
		span.End()
	}()

	if connBeginTx, ok := c.parent.(driver.ConnBeginTx); ok {
		tx, err = connBeginTx.BeginTx(ctx, opts)
		setSpanStatus(span, c.options, err)
		if err != nil {
			return nil, err
		}
		return otelTx{parent: tx, ctx: ctx, options: c.options}, nil
	}

	attrs = append(
		attrs,
		attrDeprecated,
		attribute.String(
			"sql.deprecated", "driver does not support ConnBeginTx",
		),
	)
	tx, err = c.parent.Begin()
	setSpanStatus(span, c.options, err)
	if err != nil {
		return nil, err
	}
	return otelTx{parent: tx, ctx: ctx, options: c.options}, nil
}

func (c *otelConn) CheckNamedValue(nv *driver.NamedValue) (err error) {
	nvc, ok := c.parent.(driver.NamedValueChecker)
	if ok {
		return nvc.CheckNamedValue(nv)
	}
	nv.Value, err = driver.DefaultParameterConverter.ConvertValue(nv.Value)
	return err
}
