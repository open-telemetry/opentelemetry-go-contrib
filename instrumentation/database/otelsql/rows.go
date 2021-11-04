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
	"io"
	"reflect"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Rows                           = otelRows{}
	_ driver.RowsColumnTypeDatabaseTypeName = otelRows{}
	_ driver.RowsColumnTypeLength           = otelRows{}
	_ driver.RowsColumnTypeNullable         = otelRows{}
	_ driver.RowsColumnTypePrecisionScale   = otelRows{}
	// Currently, the one exception is RowsColumnTypeScanType which does not have a
	// valid zero value. This interface is tested for and only enabled in case the
	// parent implementation supports it.
	//_ driver.RowsColumnTypeScanType         = otelRows{}
	_ driver.RowsNextResultSet = otelRows{}
)

// withRowsColumnTypeScanType is the same as the driver.RowsColumnTypeScanType
// interface except it omits the driver.Rows embedded interface.
// If the original driver.Rows implementation wrapped by otelsql supports
// RowsColumnTypeScanType we enable the original method implementation in the
// returned driver.Rows from wrapRows by doing a composition with otelRows.
type withRowsColumnTypeScanType interface {
	ColumnTypeScanType(index int) reflect.Type
}

// otelRows implements driver.Rows and all enhancement interfaces except
// driver.RowsColumnTypeScanType.
type otelRows struct {
	parent  driver.Rows
	ctx     context.Context
	options wrapper
}

//func (r otelRows) ColumnTypeScanType(index int) reflect.Type {
//	if v, ok := r.parent.(driver.RowsColumnTypeScanType); ok {
//		return v.ColumnTypeScanType(index)
//	}
//
//	return reflect.TypeOf(new(interface{}))
//}

// HasNextResultSet calls the implements the driver.RowsNextResultSet for otelRows.
// It returns the the underlying result of HasNextResultSet from the otelRows.parent
// if the parent implements driver.RowsNextResultSet.
func (r otelRows) HasNextResultSet() bool {
	if v, ok := r.parent.(driver.RowsNextResultSet); ok {
		return v.HasNextResultSet()
	}

	return false
}

// NextResultSet calls the implements the driver.RowsNextResultSet for otelRows.
// It returns the the underlying result of NextResultSet from the otelRows.parent
// if the parent implements driver.RowsNextResultSet.
func (r otelRows) NextResultSet() error {
	if v, ok := r.parent.(driver.RowsNextResultSet); ok {
		return v.NextResultSet()
	}

	return io.EOF
}

// ColumnTypeDatabaseTypeName calls the implements the driver.RowsColumnTypeDatabaseTypeName for otelRows.
// It returns the the underlying result of ColumnTypeDatabaseTypeName from the otelRows.parent
// if the parent implements driver.RowsColumnTypeDatabaseTypeName.
func (r otelRows) ColumnTypeDatabaseTypeName(index int) string {
	if v, ok := r.parent.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return v.ColumnTypeDatabaseTypeName(index)
	}

	return ""
}

// ColumnTypeLength calls the implements the driver.RowsColumnTypeLength for otelRows.
// It returns the the underlying result of ColumnTypeLength from the otelRows.parent
// if the parent implements driver.RowsColumnTypeLength.
func (r otelRows) ColumnTypeLength(index int) (length int64, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypeLength); ok {
		return v.ColumnTypeLength(index)
	}

	return 0, false
}

// ColumnTypeNullable calls the implements the driver.RowsColumnTypeNullable for otelRows.
// It returns the the underlying result of ColumnTypeNullable from the otelRows.parent
// if the parent implements driver.RowsColumnTypeNullable.
func (r otelRows) ColumnTypeNullable(index int) (nullable, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypeNullable); ok {
		return v.ColumnTypeNullable(index)
	}

	return false, false
}

// ColumnTypePrecisionScale calls the implements the driver.RowsColumnTypePrecisionScale for otelRows.
// It returns the the underlying result of ColumnTypePrecisionScale from the otelRows.parent
// if the parent implements driver.RowsColumnTypePrecisionScale.
func (r otelRows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	if v, ok := r.parent.(driver.RowsColumnTypePrecisionScale); ok {
		return v.ColumnTypePrecisionScale(index)
	}

	return 0, 0, false
}

func (r otelRows) Columns() []string {
	return r.parent.Columns()
}

func (r otelRows) Close() (err error) {
	if r.options.RowsClose {
		attrs := append([]attribute.KeyValue(nil), r.options.DefaultAttributes...)
		ctx := r.ctx
		onDeferWithErr := recordCallStats("go.sql.rows.close", r.options.InstanceName)
		defer func() {
			// Invoking this function in a defer so that we can capture
			// the value of err as set on function exit.
			onDeferWithErr(ctx, err, attrs...)
		}()

		parentSpan := trace.SpanFromContext(ctx)
		if !r.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			// we already tested driver
			return r.parent.Close()
		}

		ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			setSpanStatus(span, r.options, err)
			span.SetAttributes(attrs...)
			span.End()
		}()
	}

	err = r.parent.Close()
	return
}

func (r otelRows) Next(dest []driver.Value) (err error) {
	if r.options.RowsNext {
		attrs := append([]attribute.KeyValue(nil), r.options.DefaultAttributes...)
		ctx := r.ctx
		onDeferWithErr := recordCallStats("go.sql.rows.next", r.options.InstanceName)
		defer func() {
			// Invoking this function in a defer so that we can capture
			// the value of err as set on function exit.
			onDeferWithErr(ctx, err, attrs...)
		}()

		parentSpan := trace.SpanFromContext(ctx)
		if !r.options.AllowRoot && !parentSpan.SpanContext().IsValid() {
			// we already tested driver
			return r.parent.Close()
		}

		ctx, span := otel.Tracer("").Start(ctx, "sql:exec", trace.WithSpanKind(trace.SpanKindClient))
		defer func() {
			if err == io.EOF {
				// not an error; expected to happen during iteration
				setSpanStatus(span, r.options, nil)
			} else {
				setSpanStatus(span, r.options, err)
			}
			span.SetAttributes(attrs...)
			span.End()
		}()
	}

	err = r.parent.Next(dest)
	return
}

// wrapRows returns a struct which conforms to the driver.Rows interface.
// otelRows implements all enhancement interfaces that have no effect on
// sql/database logic in case the underlying parent implementation lacks them.
// Currently, the one exception is RowsColumnTypeScanType which does not have a
// valid zero value. This interface is tested for and only enabled in case the
// parent implementation supports it.
func wrapRows(ctx context.Context, parent driver.Rows, options wrapper) driver.Rows {
	var (
		ts, hasColumnTypeScan = parent.(driver.RowsColumnTypeScanType)
	)

	r := otelRows{
		parent:  parent,
		ctx:     ctx,
		options: options,
	}

	if hasColumnTypeScan {
		return struct {
			otelRows
			withRowsColumnTypeScanType
		}{r, ts}
	}

	return r
}
