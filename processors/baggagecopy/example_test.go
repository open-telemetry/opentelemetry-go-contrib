// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy_test

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/trace"
)

func ExampleNewSpanProcessor_allKeys() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(baggagecopy.NewSpanProcessor(baggagecopy.AllowAllMembers)),
	)
}

func ExampleNewSpanProcessor_keysWithPrefix() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(
			baggagecopy.NewSpanProcessor(
				func(m baggage.Member) bool {
					return strings.HasPrefix(m.Key(), "my-key")
				},
			),
		),
	)
}

func ExampleNewSpanProcessor_keysMatchingRegex() {
	expr := regexp.MustCompile(`^key.+`)
	trace.NewTracerProvider(
		trace.WithSpanProcessor(
			baggagecopy.NewSpanProcessor(
				func(m baggage.Member) bool {
					return expr.MatchString(m.Key())
				},
			),
		),
	)
}

func ExampleNewLogProcessor_allKeys() {
	log.NewLoggerProvider(
		log.WithProcessor(baggagecopy.NewLogProcessor(baggagecopy.AllowAllMembers)),
	)
}

func ExampleNewLogProcessor_keysWithPrefix() {
	log.NewLoggerProvider(
		log.WithProcessor(
			baggagecopy.NewLogProcessor(
				func(m baggage.Member) bool {
					return strings.HasPrefix(m.Key(), "my-key")
				},
			),
		),
	)
}

func ExampleNewLogProcessor_keysMatchingRegex() {
	expr := regexp.MustCompile(`^key.+`)
	log.NewLoggerProvider(
		log.WithProcessor(
			baggagecopy.NewLogProcessor(
				func(m baggage.Member) bool {
					return expr.MatchString(m.Key())
				},
			),
		),
	)
}
