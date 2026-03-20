// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

// labeler accumulates custom attributes that are added to metrics by the
// instrumentation. It is stored in the context under a per-instrumentation key
// so that server-handler and client-transport labelers are independent, avoiding
// attribute bleed when the same context is shared across both (for example,
// when a server handler makes an outgoing HTTP request using the same context).
//
// Attributes accumulated here are intentionally write-only from the caller's
// perspective: there is no public API to read them back. For server-side
// instrumentation the attributes are read and emitted after [middleware.serveHTTP]
// calls the next handler and it returns — i.e., when the handler function exits,
// not when the client finishes reading a streaming response body. For
// client-side instrumentation they are read immediately after the base
// [http.RoundTripper.RoundTrip] returns (response headers received), before the
// response body is read. Any attributes added after those collection points are
// silently dropped.
type labeler struct {
	// mu guards attributes against concurrent adds. Although most handlers run
	// in a single goroutine, it is valid—and common—for a handler to launch
	// additional goroutines that also call [AddHandlerMetricsAttributes] or
	// [AddClientMetricsAttributes], so thread-safety is required.
	//
	// Note: goroutines that are still running after the handler returns can
	// still call add concurrently with the get() that collects attributes for
	// metric recording. The mutex prevents a data race, but any add that loses
	// that race will be silently dropped — it arrived after the collection
	// point. This is consistent with the documented lifecycle and is not a bug.
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

func (l *labeler) add(attrs ...attribute.KeyValue) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attributes = append(l.attributes, attrs...)
}

func (l *labeler) get() []attribute.KeyValue {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := make([]attribute.KeyValue, len(l.attributes))
	copy(ret, l.attributes)
	return ret
}

type (
	handlerLabelerKeyType struct{}
	clientLabelerKeyType  struct{}
)

var (
	handlerLabelerKey handlerLabelerKeyType
	clientLabelerKey  clientLabelerKeyType
)

// AddHandlerMetricsAttributes adds custom attributes to the server-side metrics
// recorded for the in-flight request.
//
// ctx must descend from the context passed to an HTTP handler that is wrapped by
// [NewHandler] or [NewMiddleware]. If ctx does not contain a handler labeler
// (e.g., the function is called before the instrumented handler has run, after
// it has returned, or with an unrelated context such as [context.Background]),
// the call is a no-op: all attrs are silently discarded with no error or
// warning. This is intentional to keep the hot path allocation-free, but it
// means missing attributes can be hard to notice at runtime. If attributes are
// not appearing in metrics, verify that ctx is the unmodified request context
// (or a child of it) obtained from [http.Request.Context] inside the handler,
// and that it has not been replaced with a fresh context.
//
// It is safe to call this function concurrently from goroutines spawned within
// the same handler invocation.
func AddHandlerMetricsAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if l, ok := ctx.Value(handlerLabelerKey).(*labeler); ok {
		l.add(attrs...)
	}
}

// AddClientMetricsAttributes adds custom attributes to the client-side metrics
// recorded for the in-flight request.
//
// ctx must descend from the context that [Transport.RoundTrip] injects into the
// outbound request, which means this function is intended to be called from
// within an [http.RoundTripper] passed as the base transport to [NewTransport].
// If ctx does not contain a client labeler (e.g., the function is called outside
// a [NewTransport]-wrapped round trip, or with an unrelated context), the call
// is a no-op: all attrs are silently discarded with no error or warning. This is
// intentional to keep the hot path allocation-free, but it means missing
// attributes can be hard to notice at runtime. If attributes are not appearing
// in metrics, verify that you are calling this function from inside the base
// [http.RoundTripper] and that the request context has not been replaced.
//
// It is safe to call this function concurrently.
func AddClientMetricsAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if l, ok := ctx.Value(clientLabelerKey).(*labeler); ok {
		l.add(attrs...)
	}
}

// Labeler is used to allow instrumented HTTP handlers to add custom attributes to
// the metrics recorded by the net/http instrumentation.
//
// Deprecated: Use [AddHandlerMetricsAttributes] to add attributes to server
// handler metrics, or [AddClientMetricsAttributes] to add attributes to client
// transport metrics.
type Labeler struct {
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// Add attributes to a Labeler.
//
// Deprecated: Use [AddHandlerMetricsAttributes] to add attributes to server
// handler metrics, or [AddClientMetricsAttributes] to add attributes to client
// transport metrics.
func (l *Labeler) Add(ls ...attribute.KeyValue) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attributes = append(l.attributes, ls...)
}

// Get returns a copy of the attributes added to the Labeler.
//
// Deprecated: The new design is intentionally write-only; attributes are
// consumed internally by the instrumentation and cannot be read back. See
// [Labeler] for migration guidance.
func (l *Labeler) Get() []attribute.KeyValue {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := make([]attribute.KeyValue, len(l.attributes))
	copy(ret, l.attributes)
	return ret
}

type labelerContextKeyType int

const labelerContextKey labelerContextKeyType = 0

// ContextWithLabeler returns a new context with the provided Labeler instance.
// Attributes added to the specified labeler will be injected into metrics
// emitted by the instrumentation. Only one labeller can be injected into the
// context. Injecting it multiple times will override the previous calls.
//
// Deprecated: Use [AddHandlerMetricsAttributes] or [AddClientMetricsAttributes]
// instead.
func ContextWithLabeler(parent context.Context, l *Labeler) context.Context {
	return context.WithValue(parent, labelerContextKey, l)
}

// LabelerFromContext retrieves a Labeler instance from the provided context if
// one is available.  If no Labeler was found in the provided context a new, empty
// Labeler is returned and the second return value is false.  In this case it is
// safe to use the Labeler but any attributes added to it will not be used.
//
// Deprecated: Use [AddHandlerMetricsAttributes] or [AddClientMetricsAttributes]
// instead.
func LabelerFromContext(ctx context.Context) (*Labeler, bool) {
	l, ok := ctx.Value(labelerContextKey).(*Labeler)
	if !ok {
		l = &Labeler{}
	}
	return l, ok
}
