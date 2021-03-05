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

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type mockRows struct {
	shouldError bool

	closeCount, nextCount int
	nextDest              []driver.Value
}

func (m *mockRows) Columns() []string {
	return nil
}

func (m *mockRows) Close() error {
	m.closeCount++
	if m.shouldError {
		return errors.New("close")
	}
	return nil
}

func (m *mockRows) Next(dest []driver.Value) error {
	m.nextDest = dest
	m.nextCount++
	if m.shouldError {
		return errors.New("next")
	}
	return nil
}

func newMockRows(shouldError bool) *mockRows {
	return &mockRows{shouldError: shouldError}
}

var (
	_ driver.Rows = (*mockRows)(nil)
)

func TestOtRows_Close(t *testing.T) {
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

			mr := newMockRows(tc.error)
			cfg := newMockConfig(tracer)

			// New rows
			rows := newRows(context.Background(), mr, cfg)
			// Close
			err := rows.Close()

			spanList := sr.Completed()
			// A span created in newRows()
			require.Equal(t, 1, len(spanList))
			span := spanList[0]
			assert.True(t, span.Ended())

			assert.Equal(t, 1, mr.closeCount)
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.StatusCode())
				assert.Len(t, span.Events(), 1)
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.StatusCode())
			}
		})
	}
}

func TestOtRows_Next(t *testing.T) {
	testCases := []struct {
		name           string
		error          bool
		rowsNextOption bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
		{
			name:           "with RowsNextOption",
			rowsNextOption: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare traces
			sr, provider := newTracerProvider()
			tracer := provider.Tracer("test")

			mr := newMockRows(tc.error)
			cfg := newMockConfig(tracer)
			cfg.SpanOptions.RowsNext = tc.rowsNextOption

			// New rows
			rows := newRows(context.Background(), mr, cfg)
			// Next
			err := rows.Next([]driver.Value{"test"})

			spanList := sr.Started()
			// A span created in newRows()
			require.Equal(t, 1, len(spanList))
			span := spanList[0]
			assert.False(t, span.Ended())

			assert.Equal(t, 1, mr.nextCount)
			assert.Equal(t, []driver.Value{"test"}, mr.nextDest)
			var expectedEventCount int
			if tc.error {
				require.Error(t, err)
				assert.Equal(t, codes.Error, span.StatusCode())
				expectedEventCount++
			} else {
				require.NoError(t, err)
				assert.Equal(t, codes.Unset, span.StatusCode())
			}

			if tc.rowsNextOption {
				expectedEventCount++
			}
			assert.Len(t, span.Events(), expectedEventCount)
		})
	}
}

func TestNewRows(t *testing.T) {
	// Prepare traces
	sr, provider := newTracerProvider()
	tracer := provider.Tracer("test")
	ctx, dummySpan := createDummySpan(context.Background(), tracer)

	mr := newMockRows(false)
	cfg := newMockConfig(tracer)

	// New rows
	rows := newRows(ctx, mr, cfg)

	spanList := sr.Started()
	// One dummy span and one span created in newRows()
	require.Equal(t, 2, len(spanList))
	span := spanList[1]
	assert.False(t, span.Ended())
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())
	assert.Equal(t, attributesListToMap(cfg.Attributes), span.Attributes())
	assert.Equal(t, string(MethodRows), span.Name())
	assert.Equal(t, dummySpan.SpanContext().TraceID, span.SpanContext().TraceID)
	assert.Equal(t, dummySpan.SpanContext().SpanID, span.ParentSpanID())
	assert.Equal(t, mr, rows.Rows)
}
