package baggagetrace_test

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func ExampleNew_allKeys() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(baggagetrace.New(baggagetrace.AllowAllBaggageKeys)),
	)
}

func ExampleNew_keysWithPrefix() {
	trace.NewTracerProvider(
		trace.WithSpanProcessor(
			baggagetrace.New(
				func(baggageKey string) bool {
					return strings.HasPrefix(baggageKey, "my-key")
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
				func(baggageKey string) bool {
					return expr.MatchString(baggageKey)
				},
			),
		),
	)
}
