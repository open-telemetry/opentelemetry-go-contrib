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
//go:build !go1.9
// +build !go1.9

package otelsql

import (
	"database/sql/driver"
	"errors"
)

// Dummy error for setSpanStatus (does exist as sql.ErrConnDone in 1.9+)
var errConnDone = errors.New("database/sql: connection is already closed")

// otelDriver implements driver.Driver
type otelDriver struct {
	parent  driver.Driver
	options wrapper
}

func wrapDriver(d driver.Driver, o wrapper) driver.Driver {
	return otelDriver{parent: d, options: o}
}

func wrapConn(c driver.Conn, options wrapper) driver.Conn {
	return &otelConn{parent: c, options: options}
}

func wrapStmt(stmt driver.Stmt, query string, options wrapper) driver.Stmt {
	s := otelStmt{parent: stmt, query: query, options: options}
	_, hasExeCtx := stmt.(driver.StmtExecContext)
	_, hasQryCtx := stmt.(driver.StmtQueryContext)
	c, hasColCnv := stmt.(driver.ColumnConverter)
	switch {
	case !hasExeCtx && !hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
		}{s}
	case !hasExeCtx && hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtQueryContext
		}{s, s}
	case hasExeCtx && !hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
		}{s, s}
	case hasExeCtx && hasQryCtx && !hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.StmtQueryContext
		}{s, s, s}
	case !hasExeCtx && !hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.ColumnConverter
		}{s, c}
	case !hasExeCtx && hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtQueryContext
			driver.ColumnConverter
		}{s, s, c}
	case hasExeCtx && !hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.ColumnConverter
		}{s, s, c}
	case hasExeCtx && hasQryCtx && hasColCnv:
		return struct {
			driver.Stmt
			driver.StmtExecContext
			driver.StmtQueryContext
			driver.ColumnConverter
		}{s, s, s, c}
	}
	panic("unreachable")
}
