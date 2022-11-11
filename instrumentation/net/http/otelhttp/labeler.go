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

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

type labelerContextKeyType int

// LabelerContextKey is the key in the context.Context on which the Labeler instance would be placed.
const LabelerContextKey labelerContextKeyType = 0

// Labeler provides a way to hook custom attribute.KeyValue entries to a request context during the execution of the
// request. The instance of the custom Labeler can be accessed from the request's context with the LabelerContextKey.
type Labeler interface {
	Get() []attribute.KeyValue
}

// StandardLabeler is used to allow instrumented HTTP handlers to add custom attributes to
// the metrics recorded by the net/http instrumentation.
type StandardLabeler struct {
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// NewStandardLabeler returns a StandardLabeler.
func NewStandardLabeler() Labeler {
	return &StandardLabeler{}
}

// Add attributes to a StandardLabeler.
func (l *StandardLabeler) Add(ls ...attribute.KeyValue) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attributes = append(l.attributes, ls...)
}

// Get returns a copy of the attributes added to the StandardLabeler.
func (l *StandardLabeler) Get() []attribute.KeyValue {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := make([]attribute.KeyValue, len(l.attributes))
	copy(ret, l.attributes)
	return ret
}

func injectLabeler(ctx context.Context, l Labeler) context.Context {
	return context.WithValue(ctx, LabelerContextKey, l)
}

// LabelerFromContext retrieves a StandardLabeler instance from the provided context if
// one is available.  If no StandardLabeler was found in the provided context a new, empty
// StandardLabeler is returned and the second return value is false.  In this case it is
// safe to use the StandardLabeler but any attributes added to it will not be used.
func LabelerFromContext(ctx context.Context) (*StandardLabeler, bool) {
	l, ok := ctx.Value(LabelerContextKey).(*StandardLabeler)
	if !ok {
		l = &StandardLabeler{}
	}
	return l, ok
}
