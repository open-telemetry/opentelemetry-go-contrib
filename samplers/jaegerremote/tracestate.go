// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote

import (
	"strings"

	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// traceStateKey is the OpenTelemetry tracestate entry, which holds the
	// "th" threshold and "rv" randomness sub-keys.
	traceStateKey = "ot"

	thPrefix = "th:"
)

// clearTraceStateTh removes the "th" sub-key from the "ot" tracestate entry,
// keeping other sub-keys and dropping the entry if nothing remains.
//
// Non-probabilistic samplers must do this so consumers do not derive an
// adjusted count from their decision.
func clearTraceStateTh(ts oteltrace.TraceState) oteltrace.TraceState {
	otts := ts.Get(traceStateKey)
	if otts == "" {
		return ts
	}

	cleared := deleteTh(otts)
	if cleared == otts {
		return ts
	}
	if cleared == "" {
		return ts.Delete(traceStateKey)
	}

	updated, err := ts.Insert(traceStateKey, cleared)
	if err != nil {
		// Unreachable: removing a sub-key cannot make the value invalid.
		return ts.Delete(traceStateKey)
	}
	return updated
}

// deleteTh returns the "ot" entry value otts without its "th" sub-key.
func deleteTh(otts string) string {
	if !strings.HasPrefix(otts, thPrefix) && !strings.Contains(otts, ";"+thPrefix) {
		return otts
	}

	var b strings.Builder
	b.Grow(len(otts))
	for rest := otts; rest != ""; {
		var sub string
		sub, rest, _ = strings.Cut(rest, ";")
		if sub == "" || strings.HasPrefix(sub, thPrefix) {
			continue
		}
		if b.Len() > 0 {
			_, _ = b.WriteString(";")
		}
		_, _ = b.WriteString(sub)
	}
	return b.String()
}
