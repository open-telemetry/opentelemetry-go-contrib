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
	"errors"
	"io"
	"reflect"
	"testing"
)

var errDummy = errors.New("dummy")

type stubRows struct{}

func (stubRows) Columns() []string                                  { return []string{"dummy"} }
func (stubRows) Close() error                                       { return errDummy }
func (stubRows) Next([]driver.Value) error                          { return errDummy }
func (stubRows) HasNextResultSet() bool                             { return true }
func (stubRows) NextResultSet() error                               { return errDummy }
func (stubRows) ColumnTypeScanType(int) reflect.Type                { return reflect.TypeOf(stubRows{}) }
func (stubRows) ColumnTypeDatabaseTypeName(index int) string        { return "dummy" }
func (stubRows) ColumnTypeLength(index int) (length int64, ok bool) { return 1, true }
func (stubRows) ColumnTypeNullable(index int) (nullable, ok bool)   { return true, true }
func (stubRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	return 1, 1, true
}

func TestWrappingTransparency(t *testing.T) {
	var (
		ctx   = context.Background()
		oRows = &stubRows{}
		wRows = wrapRows(ctx, oRows, AllWrapperOptions)
	)

	if want, have := oRows.Columns(), wRows.Columns(); len(want) != len(have) {
		t.Errorf("rows.Column want: %v, have: %v", want, have)
	}

	if want, have := oRows.Close(), wRows.Close(); want != have {
		t.Errorf("rows.Close want: %v, have: %v", want, have)
	}

	if want, have := oRows.Next(nil), wRows.Next(nil); want != have {
		t.Errorf("rows.Next want: %v, have: %v", want, have)
	}

	if want, have := oRows.HasNextResultSet(), wRows.(driver.RowsNextResultSet).HasNextResultSet(); want != have {
		t.Errorf("rows.HasNextResultSet want: %t, have: %t", want, have)
	}

	if want, have := oRows.NextResultSet(), wRows.(driver.RowsNextResultSet).NextResultSet(); want != have {
		t.Errorf("rows.NextResultSet want: %v, have: %v", want, have)
	}

	if want, have := oRows.ColumnTypeScanType(1), wRows.(driver.RowsColumnTypeScanType).ColumnTypeScanType(1); want != have {
		t.Errorf("rows.ColumnTypeScanType want: %v, have: %v", want, have)
	}

	if want, have := oRows.ColumnTypeDatabaseTypeName(1), wRows.(driver.RowsColumnTypeDatabaseTypeName).ColumnTypeDatabaseTypeName(1); want != have {
		t.Errorf("rows.ColumnTypeDatabaseTypeName want: %s, have: %s", want, have)
	}

	oLength, oOk := oRows.ColumnTypeLength(1)
	wLength, wOk := wRows.(driver.RowsColumnTypeLength).ColumnTypeLength(1)
	if oLength != wLength || oOk != wOk {
		t.Errorf("rows.ColumnTypeLength want: %d:%t, have %d:%t", oLength, oOk, wLength, wOk)
	}

	oNullable, oOk := oRows.ColumnTypeNullable(1)
	wNullable, wOk := wRows.(driver.RowsColumnTypeNullable).ColumnTypeNullable(1)
	if oNullable != wNullable || oOk != wOk {
		t.Errorf("rows.ColumnTypeNullable want: %t:%t, have %t:%t", oNullable, oOk, wNullable, wOk)
	}

	oPrecision, oScale, oOk := oRows.ColumnTypePrecisionScale(1)
	wPrecision, wScale, wOk := wRows.(driver.RowsColumnTypePrecisionScale).ColumnTypePrecisionScale(1)
	if oPrecision != wPrecision || oScale != wScale || oOk != wOk {
		t.Errorf("rows.ColumnTypePrecisionScale want: %d:%d:%t, have %d:%d:%t", oPrecision, oScale, oOk, wPrecision, wScale, wOk)
	}
}

func TestWrappingFallback(t *testing.T) {
	var (
		ctx   = context.Background()
		oRows = struct{ driver.Rows }{&stubRows{}}
		wRows = wrapRows(ctx, oRows, AllWrapperOptions)
	)

	if want, have := oRows.Columns(), wRows.Columns(); len(want) != len(have) {
		t.Errorf("rows.Column want: %v, have: %v", want, have)
	}

	if want, have := oRows.Close(), wRows.Close(); want != have {
		t.Errorf("rows.Close want: %v, have: %v", want, have)
	}

	if want, have := oRows.Next(nil), wRows.Next(nil); want != have {
		t.Errorf("rows.Next want: %v, have: %v", want, have)
	}

	if want, have := false, wRows.(driver.RowsNextResultSet).HasNextResultSet(); want != have {
		t.Errorf("rows.HasNextResultSet want: %t, have: %t", want, have)
	}

	if want, have := io.EOF, wRows.(driver.RowsNextResultSet).NextResultSet(); want != have {
		t.Errorf("rows.NextResultSet want: %v, have: %v", want, have)
	}

	if _, ok := wRows.(driver.RowsColumnTypeScanType); ok {
		t.Error("rows.ColumnTypeScanType unexpected interface implementation found")
	}

	if want, have := "", wRows.(driver.RowsColumnTypeDatabaseTypeName).ColumnTypeDatabaseTypeName(1); want != have {
		t.Errorf("rows.ColumnTypeDatabaseTypeName want: %s, have: %s", want, have)
	}

	oLength, oOk := int64(0), false
	wLength, wOk := wRows.(driver.RowsColumnTypeLength).ColumnTypeLength(1)
	if oLength != wLength || oOk != wOk {
		t.Errorf("rows.ColumnTypeLength want: %d:%t, have %d:%t", oLength, oOk, wLength, wOk)
	}

	oNullable, oOk := false, false
	wNullable, wOk := wRows.(driver.RowsColumnTypeNullable).ColumnTypeNullable(1)
	if oNullable != wNullable || oOk != wOk {
		t.Errorf("rows.ColumnTypeNullable want: %t:%t, have %t:%t", oNullable, oOk, wNullable, wOk)
	}

	oPrecision, oScale, oOk := int64(0), int64(0), false
	wPrecision, wScale, wOk := wRows.(driver.RowsColumnTypePrecisionScale).ColumnTypePrecisionScale(1)
	if oPrecision != wPrecision || oScale != wScale || oOk != wOk {
		t.Errorf("rows.ColumnTypePrecisionScale want: %d:%d:%t, have %d:%d:%t", oPrecision, oScale, oOk, wPrecision, wScale, wOk)
	}
}
