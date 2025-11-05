// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

// Labeler is used to allow instrumented gRPC handlers to add custom attributes to
// the metrics recorded by the gRPC instrumentation.
type Labeler struct {
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// Add attributes to a Labeler.
func (l *Labeler) Add(ls ...attribute.KeyValue) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attributes = append(l.attributes, ls...)
}

// Get returns a copy of the attributes added to the Labeler.
func (l *Labeler) Get() []attribute.KeyValue {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := make([]attribute.KeyValue, len(l.attributes))
	copy(ret, l.attributes)

	return ret
}

// LabelerDirection indicates whether the Labeler applies to the client or to the server.
type LabelerDirection int

const (
	// ClientLabelerDirection specifies that the labeler applies to the client.
	ClientLabelerDirection LabelerDirection = iota
	// ServerLabelerDirection specifies that the labeler applies to the server.
	ServerLabelerDirection
)

type labelerContextKeyType string

const (
	clientContextKey labelerContextKeyType = "otelgrpc.client.labeler"
	serverContextKey labelerContextKeyType = "otelgrpc.server.labeler"
)

// ContextWithLabeler returns a new context with the provided Labeler instance.
// Attributes added to the Labeler will be injected into metrics
// emitted by the instrumentation associated with the specified LabelerDirection.
//
// Only one Labeler can be injected for the same direction.
// Injecting a Labeler for the same direction will override the previous call.
func ContextWithLabeler(parent context.Context, l *Labeler, direction LabelerDirection) context.Context {
	switch direction {
	case ClientLabelerDirection:
		return context.WithValue(parent, clientContextKey, l)
	case ServerLabelerDirection:
		return context.WithValue(parent, serverContextKey, l)
	default:
		return parent
	}
}

// LabelerFromContext retrieves a Labeler instance from the provided context if one is available for
// the specified LabelerDirection.
//
// If no Labeler was found in the provided context a new, empty Labeler
// for the specified direction is returned and the second return value is false.
// In this case it is safe to use the Labeler but any attributes added to it will not be used.
func LabelerFromContext(ctx context.Context, direction LabelerDirection) (*Labeler, bool) {
	var key labelerContextKeyType
	switch direction {
	case ClientLabelerDirection:
		key = clientContextKey
	case ServerLabelerDirection:
		key = serverContextKey
	default:
		return &Labeler{}, false
	}

	l, ok := ctx.Value(key).(*Labeler)
	if ok {
		return l, true
	}

	return &Labeler{}, false
}
