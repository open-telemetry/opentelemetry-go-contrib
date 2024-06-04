// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package baggagetrace implements a baggage span processor.
//
// This is an OpenTelemetry [Span Processor] that reads key/values stored in
// [Baggage] in the starting span's parent context and adds them as attributes
// to the span.
//
// Keys and values added to Baggage will appear on all subsequent child spans for
// a trace within this service *and* will be propagated to external services via
// propagation headers.
// If the external services also have a Baggage span processor, the keys and
// values will appear in those child spans as well.
//
// ⚠️ Warning ⚠️
// To repeat: a consequence of adding data to Baggage is that the keys and values
// will appear in all outgoing HTTP baggage headers from the application.
//
// Do not put sensitive information in Baggage.
//
// # Usage
//
// Add the span processor when configuring the tracer provider.
//
// The convience function `baggagetrace.AllowAllBaggageKeys` is provided to
// allow all baggage keys to be copied to the span. Alternatively, you can
// provide a custom baggage key predicate to select which baggage keys you want
// to copy.
//
// For example, to use the convience `baggagetrace.AllowAllBaggageKeys` to copy
// all baggage entries:
//
//	import (
//		"go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
//	)
//
//	tp := trace.NewTracerProvider(
//		trace.WithSpanProcessor(baggagetrace.New(baggagetrace.AllowAllBaggageKeys)),
//		// ...
//	)
//
// Alternatively, you can provide a custom baggage key predicate to select
// which baggage keys you want to copy.
//
// For example, to only copy baggage entries that start with 'my-key':
//
//	baggagetrace.New(func(baggageKey string) bool {
//		return strings.HasPrefix(baggageKey, "my-key")
//	})
//
// For example, to only copy baggage entries that match the regex '^key.+':
//
//	expr := regexp.MustCompile(`^key.+`)
//
//	baggagetrace.New(func(baggageKey string) bool {
//		return expr.MatchString(baggageKey)
//	})
//
// [Span Processor]: https://opentelemetry.io/docs/specs/otel/trace/sdk/#span-processor
// [Baggage]: https://opentelemetry.io/docs/specs/otel/api/baggage
package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
