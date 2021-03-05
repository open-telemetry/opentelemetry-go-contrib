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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

type mockStmt struct {
	driver.Stmt

	shouldError bool
	queryCount  int
	execCount   int

	queryContextArgs []driver.NamedValue
	ExecContextArgs  []driver.NamedValue
}

func newMockStmt(shouldError bool) *mockStmt {
	return &mockStmt{shouldError: shouldError}
}

func (m *mockStmt) CheckNamedValue(value *driver.NamedValue) error {
	if m.shouldError {
		return errors.New("checkNamedValue")
	}
	return nil
}

func (m *mockStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	m.queryContextArgs = args
	m.queryCount++
	if m.shouldError {
		return nil, errors.New("queryContext")
	}
	return nil, nil
}

func (m *mockStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	m.ExecContextArgs = args
	m.execCount++
	if m.shouldError {
		return nil, errors.New("execContext")
	}
	return nil, nil
}

var (
	_ driver.Stmt              = (*mockStmt)(nil)
	_ driver.StmtExecContext   = (*mockStmt)(nil)
	_ driver.StmtQueryContext  = (*mockStmt)(nil)
	_ driver.NamedValueChecker = (*mockStmt)(nil)
)

func TestOtStmt_ExecContext(t *testing.T) {
	testCases := []struct {
		name  string
		error bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			sr, provider := newTracerProvider()
			tracer := provider.Tracer("test")
			ctx, dummySpan := createDummySpan(context.Background(), tracer)
			ms := newMockStmt(tc.error)

			// New stmt
			cfg := newMockConfig(tracer)
			stmt := newStmt(ms, cfg, "query")
			// Exec
			_, err := stmt.ExecContext(ctx, []driver.NamedValue{{Name: "test"}})

			spanList := sr.Completed()
			// One dummy span and a span created in tx
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.True(t, span.Ended())
			assert.Equal(t, trace.SpanKindClient, span.SpanKind())
			assert.Equal(t, attributesListToMap(append([]attribute.KeyValue{semconv.DBStatementKey.String("query")},
				cfg.Attributes...)), span.Attributes())
			assert.Equal(t, string(MethodStmtExec), span.Name())
			assert.Equal(t, dummySpan.SpanContext().TraceID, span.SpanContext().TraceID)
			assert.Equal(t, dummySpan.SpanContext().SpanID, span.ParentSpanID())

			assert.Equal(t, 1, ms.execCount)
			assert.Equal(t, []driver.NamedValue{{Name: "test"}}, ms.ExecContextArgs)
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.StatusCode())
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.StatusCode())
			}
		})
	}
}

func TestOtStmt_QueryContext(t *testing.T) {
	testCases := []struct {
		name  string
		error bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			sr, provider := newTracerProvider()
			tracer := provider.Tracer("test")
			ctx, dummySpan := createDummySpan(context.Background(), tracer)
			ms := newMockStmt(tc.error)

			// New stmt
			cfg := newMockConfig(tracer)
			stmt := newStmt(ms, cfg, "query")
			// Query
			rows, err := stmt.QueryContext(ctx, []driver.NamedValue{{Name: "test"}})

			spanList := sr.Completed()
			// One dummy span and a span created in tx
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.True(t, span.Ended())
			assert.Equal(t, trace.SpanKindClient, span.SpanKind())
			assert.Equal(t, attributesListToMap(append([]attribute.KeyValue{semconv.DBStatementKey.String("query")},
				cfg.Attributes...)), span.Attributes())
			assert.Equal(t, string(MethodStmtQuery), span.Name())
			assert.Equal(t, dummySpan.SpanContext().TraceID, span.SpanContext().TraceID)
			assert.Equal(t, dummySpan.SpanContext().SpanID, span.ParentSpanID())

			assert.Equal(t, 1, ms.queryCount)
			assert.Equal(t, []driver.NamedValue{{Name: "test"}}, ms.queryContextArgs)
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.StatusCode())
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.StatusCode())
				assert.IsType(t, &otRows{}, rows)
			}
		})
	}
}
