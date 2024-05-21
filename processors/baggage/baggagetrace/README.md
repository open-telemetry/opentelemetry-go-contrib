# Baggage Span Processor

This is an OpenTelemetry [span processor](https://opentelemetry.io/docs/specs/otel/trace/sdk/#span-processor) that reads key/values stored in [Baggage](https://opentelemetry.io/docs/specs/otel/baggage/api/) in the starting span's parent context and adds them as attributes to the span.

Keys and values added to Baggage will appear on all subsequent child spans for a trace within this service *and* will be propagated to external services via propagation headers.
If the external services also have a Baggage span processor, the keys and values will appear in those child spans as well.

⚠️ Waning ⚠️
To repeat: a consequence of adding data to Baggage is that the keys and values will appear in all outgoing HTTP headers from the application.

Do not put sensitive information in Baggage.

## Usage

Add the span processor when configuring the tracer provider.

To configure the span processor to copy all baggage entries during configuration:

```golang
import (
    "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
)

tp := trace.NewTracerProvider(
    trace.WithSpanProcessor(baggagetrace.New(baggagetrace.AllowAllBaggageKeys)),
    // ...
)
```

Alternatively, you can provide a custom baggage key predicate to select which baggage keys you want to copy.

For example, to only copy baggage entries that start with `my-key`:

```golang
trace.WithSpanProcessor(baggagetrace.New(func(baggageKey string) bool {
    return strings.HasPrefix(baggageKey, "my-key")
}))
```

For example, to only copy baggage entries that match the regex `^key.+`:

```golang
expr := regexp.MustCompile(`^key.+`)
trace.WithSpanProcessor(baggagetrace.New(func(baggageKey string) bool {
    return expr.MatchString(baggageKey)
}))
```
