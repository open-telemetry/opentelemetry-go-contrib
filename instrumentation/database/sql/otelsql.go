package otelsql

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/ngrok/sqlmw"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/database/sql"
)

func Register(drivername string, dri driver.Driver, opts ...Option) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.traceProvider == nil {
		cfg.traceProvider = global.TraceProvider()
	}
	sqlInt := &sqlInterceptor{
		tr:              cfg.traceProvider.Tracer(defaultTracerName),
		traceAttributes: cfg.traceAttributes,
	}
	sql.Register(drivername, sqlmw.Driver(dri, sqlInt))
}

type sqlInterceptor struct {
	sqlmw.NullInterceptor
	tr              trace.Tracer
	traceAttributes traceAttributes
}

func (in *sqlInterceptor) ConnBeginTx(ctx context.Context, conn driver.ConnBeginTx, txOpts driver.TxOptions) (driver.Tx, error) {
	ctx, span := in.tr.Start(ctx, "ConnBeginTx", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return conn.BeginTx(ctx, txOpts)
}

func (in *sqlInterceptor) ConnPrepareContext(ctx context.Context, conn driver.ConnPrepareContext, query string) (driver.Stmt, error) {
	traceAttributes := append(in.traceAttributes, label.String("sql.query", query))
	ctx, span := in.tr.Start(ctx, "ConnPrepareContext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(traceAttributes...))
	defer span.End()
	return conn.PrepareContext(ctx, query)
}

func (in *sqlInterceptor) ConnPing(ctx context.Context, conn driver.Pinger) error {
	ctx, span := in.tr.Start(ctx, "ConnPing", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return conn.Ping(ctx)
}

func (in *sqlInterceptor) ConnExecContext(ctx context.Context, conn driver.ExecerContext, query string, args []driver.NamedValue) (driver.Result, error) {
	traceAttributes := append(in.traceAttributes, label.String("sql.query", query))
	ctx, span := in.tr.Start(ctx, "ConnExecContext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(traceAttributes...))
	defer span.End()
	return conn.ExecContext(ctx, query, args)
}

func (in *sqlInterceptor) ConnQueryContext(ctx context.Context, conn driver.QueryerContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	traceAttributes := append(in.traceAttributes, label.String("sql.query", query))
	ctx, span := in.tr.Start(ctx, "ConnQueryContext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(traceAttributes...))
	defer span.End()
	return conn.QueryContext(ctx, query, args)
}

func (in *sqlInterceptor) ConnectorConnect(ctx context.Context, connect driver.Connector) (driver.Conn, error) {
	ctx, span := in.tr.Start(ctx, "ConnectorConnect", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return connect.Connect(ctx)
}

func (in *sqlInterceptor) ResultLastInsertId(res driver.Result) (int64, error) {
	ctx := context.Background()
	_, span := in.tr.Start(ctx, "ResultLastInsertId", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return res.LastInsertId()
}

func (in *sqlInterceptor) ResultRowsAffected(res driver.Result) (int64, error) {
	ctx := context.Background()
	_, span := in.tr.Start(ctx, "ResultRowsAffected", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return res.RowsAffected()
}

func (in *sqlInterceptor) RowsNext(ctx context.Context, rows driver.Rows, dest []driver.Value) error {
	_, span := in.tr.Start(ctx, "RowsNext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return rows.Next(dest)
}

func (in *sqlInterceptor) StmtExecContext(ctx context.Context, stmt driver.StmtExecContext, _ string, args []driver.NamedValue) (driver.Result, error) {
	ctx, span := in.tr.Start(ctx, "StmtExecContext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return stmt.ExecContext(ctx, args)
}

func (in *sqlInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, _ string, args []driver.NamedValue) (driver.Rows, error) {
	ctx, span := in.tr.Start(ctx, "StmtQueryContext", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return stmt.QueryContext(ctx, args)
}

func (in *sqlInterceptor) StmtClose(ctx context.Context, stmt driver.Stmt) error {
	_, span := in.tr.Start(ctx, "StmtClose", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return stmt.Close()
}

func (in *sqlInterceptor) TxCommit(ctx context.Context, tx driver.Tx) error {
	_, span := in.tr.Start(ctx, "TxCommit", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return tx.Commit()
}

func (in *sqlInterceptor) TxRollback(ctx context.Context, tx driver.Tx) error {
	_, span := in.tr.Start(ctx, "TxRollback", trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(in.traceAttributes...))
	defer span.End()
	return tx.Rollback()
}
