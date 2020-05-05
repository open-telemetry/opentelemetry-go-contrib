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

func maybeNewStmt(realStmt driver.Stmt, setup *tracingSetup, savedQuery string) driver.Stmt {
	if realStmt == nil {
		return nil
	}
	return newStmt(realStmt, setup, savedQuery)
}

// driver.Stmt functions for driver.Stmt

func traceDSClose(r driver.Stmt, setup *tracingSetup, savedQuery string) error {
	return r.Close()
}

func traceDSNumInput(r driver.Stmt, setup *tracingSetup, savedQuery string) int {
	return r.NumInput()
}

func traceDSExec(r driver.Stmt, setup *tracingSetup, savedQuery string, args []driver.Value) (driver.Result, error) {
	ctx, span := setup.StartNoCtx("stmt exec", savedQuery)
	res, err := r.Exec(args) //nolint:staticcheck // silence the deprecation warning
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return res, err
}

func traceDSQuery(r driver.Stmt, setup *tracingSetup, savedQuery string, args []driver.Value) (driver.Rows, error) {
	ctx, span := setup.StartNoCtx("stmt query", savedQuery)
	rows, err := r.Query(args) //nolint:staticcheck // silence the deprecation warning
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return rows, err
}

// driver.Stmt functions for driver.ColumnConverter

func traceDSColumnConverter(r driver.ColumnConverter, setup *tracingSetup, savedQuery string, idx int) driver.ValueConverter { //nolint:staticcheck // silence the deprecation warning
	return r.ColumnConverter(idx)
}

// driver.Stmt functions for driver.NamedValueChecker

func traceDSCheckNamedValue(r driver.NamedValueChecker, setup *tracingSetup, savedQuery string, value *driver.NamedValue) error {
	return r.CheckNamedValue(value)
}

// driver.Stmt functions for driver.StmtExecContext

func traceDSExecContext(r driver.StmtExecContext, setup *tracingSetup, savedQuery string, ctx context.Context, args []driver.NamedValue) (driver.Result, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.Start(ctx, "stmt exec", savedQuery)
	res, err := r.ExecContext(ctx, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return res, err
}

// driver.Stmt functions for driver.StmtQueryContext

func traceDSQueryContext(r driver.StmtQueryContext, setup *tracingSetup, savedQuery string, ctx context.Context, args []driver.NamedValue) (driver.Rows, error) { //nolint:golint // context.Context is not first
	ctx, span := setup.Start(ctx, "stmt query", savedQuery)
	rows, err := r.QueryContext(ctx, args)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return rows, err
}
