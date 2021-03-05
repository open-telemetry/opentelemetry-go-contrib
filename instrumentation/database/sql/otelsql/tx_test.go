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
	"go.opentelemetry.io/otel/trace"
)

type mockTx struct {
	shouldError bool

	commitCount   int
	rollbackCount int
}

func newMockTx(shouldError bool) *mockTx {
	return &mockTx{shouldError: shouldError}
}

func (m *mockTx) Commit() error {
	m.commitCount++
	if m.shouldError {
		return errors.New("commit")
	}
	return nil
}

func (m *mockTx) Rollback() error {
	m.rollbackCount++
	if m.shouldError {
		return errors.New("rollback")
	}
	return nil
}

var _ driver.Tx = (*mockTx)(nil)

var defaultattribute = attribute.Key("test").String("foo")

func TestOtTx_Commit(t *testing.T) {
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
			mt := newMockTx(tc.error)

			// New tx
			cfg := newMockConfig(tracer)
			tx := newTx(ctx, mt, cfg)
			// Commit
			err := tx.Commit()

			spanList := sr.Completed()
			// One dummy span and one span created in tx
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.True(t, span.Ended())
			assert.Equal(t, trace.SpanKindClient, span.SpanKind())
			assert.Equal(t, attributesListToMap(cfg.Attributes), span.Attributes())
			assert.Equal(t, string(MethodTxCommit), span.Name())
			assert.Equal(t, dummySpan.SpanContext().TraceID, span.SpanContext().TraceID)
			assert.Equal(t, dummySpan.SpanContext().SpanID, span.ParentSpanID())

			assert.Equal(t, 1, mt.commitCount)
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

func TestOtTx_Rollback(t *testing.T) {
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
			mt := newMockTx(tc.error)

			// New tx
			cfg := newMockConfig(tracer)
			tx := newTx(ctx, mt, cfg)
			// Rollback
			err := tx.Rollback()

			spanList := sr.Completed()
			// One dummy span and a span created in tx
			require.Equal(t, 2, len(spanList))
			span := spanList[1]
			assert.True(t, span.Ended())
			assert.Equal(t, trace.SpanKindClient, span.SpanKind())
			assert.Equal(t, attributesListToMap(cfg.Attributes), span.Attributes())
			assert.Equal(t, string(MethodTxRollback), span.Name())
			assert.Equal(t, dummySpan.SpanContext().TraceID, span.SpanContext().TraceID)
			assert.Equal(t, dummySpan.SpanContext().SpanID, span.ParentSpanID())
			assert.Equal(t, 1, mt.rollbackCount)

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
