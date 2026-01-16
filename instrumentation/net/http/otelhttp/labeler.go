// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

// Deprecated: Labeler is deprecated and will be removed in a future release.
// Use WithMetricAttributesFn instead to supply custom metric attributes.
//
// Migration example:
//
//	// Before:
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		labeler, _ := otelhttp.LabelerFromContext(r.Context())
//		labeler.Add(attribute.String("user.id", getUserID(r)))
//	})
//	handler = otelhttp.NewHandler(handler, "operation")
//
//	// After:
//	handler := otelhttp.NewHandler(handler, "operation",
//		otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
//			return []attribute.KeyValue{
//				attribute.String("user.id", getUserID(r)),
//			}
//		}),
//	)
type Labeler struct {
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// Deprecated: Use WithMetricAttributesFn instead.
func (l *Labeler) Add(ls ...attribute.KeyValue) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attributes = append(l.attributes, ls...)
}

// Deprecated: Use WithMetricAttributesFn instead.
func (l *Labeler) Get() []attribute.KeyValue {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := make([]attribute.KeyValue, len(l.attributes))
	copy(ret, l.attributes)
	return ret
}

type labelerContextKeyType int

const labelerContextKey labelerContextKeyType = 0

// Deprecated: Use WithMetricAttributesFn instead.
func ContextWithLabeler(parent context.Context, l *Labeler) context.Context {
	return context.WithValue(parent, labelerContextKey, l)
}

// Deprecated: Use WithMetricAttributesFn instead.
func LabelerFromContext(ctx context.Context) (*Labeler, bool) {
	l, ok := ctx.Value(labelerContextKey).(*Labeler)
	if !ok {
		l = &Labeler{}
	}
	return l, ok
}
