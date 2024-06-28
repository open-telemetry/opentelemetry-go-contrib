// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagetrace_test

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

func ExampleNew_allKeys() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(baggagetrace.New(baggagetrace.AllowAllMembers)),
	)
}

func ExampleNew_keysWithPrefix() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(
			baggagetrace.New(
				func(m baggage.Member) bool {
					return strings.HasPrefix(m.Key(), "my-key")
				},
			),
		),
	)
}

func ExampleNew_keysMatchingRegex() {
	expr := regexp.MustCompile(`^key.+`)
	trace.NewTracerProvider(
		trace.WithSpanProcessor(
			baggagetrace.New(
				func(m baggage.Member) bool {
					return expr.MatchString(m.Key())
				},
			),
		),
	)
}
