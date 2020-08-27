package http

import (
	"context"

	"go.opentelemetry.io/otel/label"
)

// Labeler is used to allow instrumented HTTP handlers to add custom labels to
// the metrics recorded by the net/http instrumentation.
type Labeler struct {
	labels []label.KeyValue
}

// Add labels to a Labeler.
func (l *Labeler) Add(ls ...label.KeyValue) {
	l.labels = append(l.labels, ls...)
}

type labelerContextKeyType int

const lablelerContextKey labelerContextKeyType = 0

func injectLabeler(ctx context.Context, l *Labeler) context.Context {
	return context.WithValue(ctx, lablelerContextKey, l)
}

// LabelerFromContext retrieves a Labeler instance from the provided context if
// one is available.  If no labeler was found in the provided context a new, empty
// Labeler is returned and the second return value is false.  In this case it is
// safe to use the Labeler but any labels added to it will not be used.
func LabelerFromContext(ctx context.Context) (*Labeler, bool) {
	l, ok := ctx.Value(lablelerContextKey).(*Labeler)
	if !ok {
		l = &Labeler{}
	}
	return l, ok
}
