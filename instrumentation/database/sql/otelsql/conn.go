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

	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

var (
	_ driver.Pinger             = (*otConn)(nil)
	_ driver.Execer             = (*otConn)(nil) // nolint
	_ driver.ExecerContext      = (*otConn)(nil)
	_ driver.Queryer            = (*otConn)(nil) // nolint
	_ driver.QueryerContext     = (*otConn)(nil)
	_ driver.Conn               = (*otConn)(nil)
	_ driver.ConnPrepareContext = (*otConn)(nil)
	_ driver.ConnBeginTx        = (*otConn)(nil)
	_ driver.SessionResetter    = (*otConn)(nil)
	_ driver.NamedValueChecker  = (*otConn)(nil)
)

type otConn struct {
	driver.Conn
	otDriver *otDriver
	cfg      config
}

func newConn(conn driver.Conn, otDriver *otDriver) *otConn {
	return &otConn{
		Conn:     conn,
		otDriver: otDriver,
		cfg:      otDriver.cfg,
	}
}

func (c *otConn) Ping(ctx context.Context) (err error) {
	pinger, ok := c.Conn.(driver.Pinger)
	if !ok {
		return driver.ErrSkip
	}

	if c.otDriver.cfg.SpanOptions.Ping {
		var span trace.Span
		ctx, span = c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnPing, ""),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(c.cfg.Attributes...),
		)
		defer func() {
			if err != nil {
				recordSpanError(span, c.cfg.SpanOptions, err)
			}
			span.End()
		}()
	}

	err = pinger.Ping(ctx)
	return err
}

func (c *otConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	execer, ok := c.Conn.(driver.Execer) // nolint
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.Exec(query, args)
}

func (c *otConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (res driver.Result, err error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	ctx, span := c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnExec, query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			append(c.cfg.Attributes,
				semconv.DBStatementKey.String(query),
			)...),
	)
	defer span.End()

	res, err = execer.ExecContext(ctx, query, args)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return nil, err
	}
	return res, nil
}

func (c *otConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.Queryer) // nolint
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.Query(query, args)
}

func (c *otConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	queryCtx, span := c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnQuery, query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			append(c.cfg.Attributes,
				semconv.DBStatementKey.String(query),
			)...),
	)
	defer span.End()

	rows, err = queryer.QueryContext(queryCtx, query, args)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return nil, err
	}
	return newRows(ctx, rows, c.cfg), nil
}

func (c *otConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	preparer, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	ctx, span := c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnPrepare, query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			append(c.cfg.Attributes,
				semconv.DBStatementKey.String(query),
			)...),
	)
	defer span.End()

	stmt, err = preparer.PrepareContext(ctx, query)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return nil, err
	}
	return newStmt(stmt, c.cfg, query), nil
}

func (c *otConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	connBeginTx, ok := c.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, driver.ErrSkip
	}

	ctx, span := c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnBeginTx, ""),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(c.cfg.Attributes...),
	)
	defer span.End()

	tx, err = connBeginTx.BeginTx(ctx, opts)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return nil, err
	}
	return newTx(ctx, tx, c.cfg), nil
}

func (c *otConn) ResetSession(ctx context.Context) (err error) {
	sessionResetter, ok := c.Conn.(driver.SessionResetter)
	if !ok {
		return driver.ErrSkip
	}

	ctx, span := c.cfg.Tracer.Start(ctx, c.cfg.SpanNameFormatter.Format(ctx, MethodConnResetSession, ""),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(c.cfg.Attributes...),
	)
	defer span.End()

	err = sessionResetter.ResetSession(ctx)
	if err != nil {
		recordSpanError(span, c.cfg.SpanOptions, err)
		return err
	}
	return nil
}

func (c *otConn) CheckNamedValue(namedValue *driver.NamedValue) error {
	namedValueChecker, ok := c.Conn.(driver.NamedValueChecker)
	if !ok {
		return driver.ErrSkip
	}

	return namedValueChecker.CheckNamedValue(namedValue)
}
