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

package driver

import (
	"context"
	"database/sql/driver"
)

func maybeNewConn(realConn driver.Conn, setup *tracingSetup) driver.Conn {
	if realConn == nil {
		return nil
	}
	return newConn(realConn, setup)
}

// driver.Conn functions for driver.Conn

func traceDCPrepare(r driver.Conn, setup *tracingSetup, query string) (driver.Stmt, error) {
	ctx, span := setup.StartNoCtx("prepare", query)
	realStmt, err := r.Prepare(query)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return maybeNewStmt(realStmt, setup, query), err
}

func traceDCClose(r driver.Conn, setup *tracingSetup) error {
	return r.Close()
}

func traceDCBegin(r driver.Conn, setup *tracingSetup) (driver.Tx, error) {
	ctx, span := setup.StartNoCtxNoStmt("transaction begin")
	realTx, err := r.Begin() //nolint:staticcheck // silence the deprecation warning
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return maybeNewTx(realTx, setup), err
}

// driver.Conn functions for driver.ConnBeginTx

func traceDCBeginTx(r driver.ConnBeginTx, setup *tracingSetup, ctx context.Context, opts driver.TxOptions) (driver.Tx, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.StartNoStmt(ctx, "transaction begin")
	realTx, err := r.BeginTx(ctx, opts)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return maybeNewTx(realTx, setup), err
}

// driver.Conn functions for driver.ConnPrepareContext

func traceDCPrepareContext(r driver.ConnPrepareContext, setup *tracingSetup, ctx context.Context, query string) (driver.Stmt, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.Start(ctx, "prepare", query)
	realStmt, err := r.PrepareContext(ctx, query)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return maybeNewStmt(realStmt, setup, query), err
}

// driver.Conn functions for driver.Execer

func traceDCExec(r driver.Execer, setup *tracingSetup, query string, args []driver.Value) (driver.Result, error) { //nolint:staticcheck // silence the deprecation warning
	ctx, span := setup.StartNoCtx("exec", query)
	res, err := r.Exec(query, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return res, err
}

// driver.Conn functions for driver.ExecerContext

func traceDCExecContext(r driver.ExecerContext, setup *tracingSetup, ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.Start(ctx, "exec", query)
	res, err := r.ExecContext(ctx, query, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return res, err
}

// driver.Conn functions for driver.NamedValueChecker

func traceDCCheckNamedValue(r driver.NamedValueChecker, setup *tracingSetup, value *driver.NamedValue) error {
	return r.CheckNamedValue(value)
}

// driver.Conn functions for driver.Pinger

func traceDCPing(r driver.Pinger, setup *tracingSetup, ctx context.Context) error { //nolint:golint // context.Context is not first
	ctx, span := setup.StartNoStmt(ctx, "ping")
	err := r.Ping(ctx)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return err
}

// driver.Conn functions for driver.Queryer

func traceDCQuery(r driver.Queryer, setup *tracingSetup, query string, args []driver.Value) (driver.Rows, error) { //nolint:staticcheck // silence the deprecation warning
	ctx, span := setup.StartNoCtx("query", query)
	rows, err := r.Query(query, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return rows, err
}

// driver.Conn functions for driver.QueryerContext

func traceDCQueryContext(r driver.QueryerContext, setup *tracingSetup, ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.Start(ctx, "query", query)
	rows, err := r.QueryContext(ctx, query, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return rows, err
}

// driver.Conn functions for driver.SessionResetter

func traceDCResetSession(r driver.SessionResetter, setup *tracingSetup, ctx context.Context) error { //nolint:golint // context.Context is not first
	return r.ResetSession(ctx)
}
