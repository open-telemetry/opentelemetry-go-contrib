# OTel HTTP Conv

## Motivation

We provide many instrumentation libraries for HTTP packages and frameworks.
Many of them need to reimplement similar features, as they have their own
internals (they don't all allow using `net/http.Handler`).
This is causing a lot of duplication across instrumentations, and makes
standardization harder.

Folks have also expressed interest in being able to use the internal tools
provided by the internal `semconvutil` package within their own
instrumentations ([#4580](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4580)).
This package aims to solve that by being publicly usable.

This document outlines a proposal for a new instrumentation module called
`otelhttpconv` that will provide a way to reduce some of this code repetition,
especially with the setup of spans and metrics and their attributes.

This package will be used by every official instrumentation that handles HTTP
requests. It may be used by external implementers as well.
This package therefore needs to provide a clear, consistent and stable API that
instrumentations can use, hence this document.

That API will have the following requirements:

* Everything it provides must be done without the external use of internal packages.
	Even though official instrumentations can technically import internal
	packages from different modules, external ones cannot. And doing so leads to
	unexpected breaking changes.
* Minimal number of breaking changes once the module is shipped.
	While we can't publish a new module as stable directly (we may have missed
	things), this module will need to become stable as soon as any of the HTTP
	instrumentations become stable.
	As our goal is to make `otelhttp` stable in 2025, stabilization should happen
	within the same timeframe.

The goal of this document is also to make future semantic convention migrations
easier, and to allow flexibility with the use or not of unstable
attributes/metrics.

## Design

The proposed design aims to:

* Work with every official instrumentation, and be available to external implementers.
* Provide flexibility in its use, to allow folks the use of unstable semantic conventions if they wish.

### Interfaces

We will provide two public interface that allow interacting with the
instrumentation package.
One for client. and one for servers. Implementations can use one, the other or both.

#### Shared between client and server

```golang
// RequestWrapper provides a layer on top of `*http.Response` that tracks
// additional information such as bytes read etc.
type ResponseWrapper interface {
	http.ResponseWriter

	// Duration contains the duration of the HTTP request
	Duration() time.Duration

	// Duration contains the status code returned by the HTTP request
	StatusCode() int

	// BytesRead contains the amount of bytes read from the response's body
	BytesRead() int64
}
```

#### The client

```golang
// Client provides an interface for HTTP client instrumentations to set the
// proper semantic convention attributes and metrics into their data.
type Client interface {
	// RecordError records an error returned by the HTTP request.
	RecordError(error)

	// RecordMetrics records the metrics from the provided HTTP request.
	RecordMetrics(ctx context.Context, w ResponseWrapper)

	// RecordSpan sets the span attributes and status code based on the HTTP
	// request and response.
	// This method does not create a new span. It retrieves the current one from
	// the context.
	// It remains the instrumentation's responsibility to start and end spans.
	RecordSpan(ctx context.Context, req *http.Request, w ResponseWrapper, cfg ...ClientRecordSpanOption)
}

// ClientRecordSpanOption applies options to the RecordSpan method
type ClientRecordSpanOption interface{}
```

#### The Server

```golang
// Server provides an interface for HTTP server instrumentations to set the
// proper semantic convention attributes and metrics into their data.
type Server interface {

	// RecordMetrics records the metrics from the provided HTTP request.
	RecordMetrics(ctx context.Context, w ResponseWrapper)

	// RecordSpan sets the span attributes and status code based on the HTTP
	// request and response.
	// This method does not create a new span. It retrieves the current one from
	// the context.
	// It remains the instrumentation's responsibility to start and end spans.
	RecordSpan(ctx context.Context, req *http.Request, w ResponseWrapper, cfg ...ServerRecordSpanOption)
}

// ServerRecordSpanOption applies options to the RecordSpan method
type ServerRecordSpanOption interface{}
```

The `ClientRecordSpanOption` and `ServerRecordSpanOption` functional options
allows passing optional parameters to be used within the `RecordSpan` method,
such as the HTTP route.

When the data those options provide is not specified, the related span
attributes will not be set.

#### Request and Response wrappers

The request and response wrappers implementations will be provided by the
[internal/request](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/request)
templates package.

We may provide a new package, or actual implementations within the
`otelhttpconv` package later on. But doing so is not covered by this design
document at the moment.

Therefore external implementations will currently have to implement this logic
themselves.

### Implementations

We will provide one official implementation of the described interfaces.
This implementation will have the following requirements:

* Support for the latest semantic conventions only.
* Support for stable semantic conventions only.

We may provide additional implementations later on such as:

* An implementation that serves as a proxy to allow combining multiple implementations together.
* An implementation that covers unstable semantic conventions.

#### Example implementation

The implementation example here is kept simple for the purpose of
understandability.

The following code sample provides a simple `http.Handler` that implements the
provided interface to instrument HTTP applications.

Because both the client and server interface are very similar, a client
implementation would be similar too.

```golang
type middleware struct {
	operation string

	tracer      trace.Tracer
	propagators propagation.TextMapPropagator
	meter       metric.Meter
	httpconv    otelhttpconv.Server
}

// NewMiddleware returns a tracing and metrics instrumentation middleware.
// The handler returned by the middleware wraps a handler
// in a span named after the operation and enriches it with metrics.
func NewMiddleware(operation string) func(http.Handler) http.Handler {
	m := middleware{
		operation:   operation,
		tracer:      otel.Tracer("http"),
		propagators: otel.GetTextMapPropagator(),
		meter:       otel.Meter("http"),
		httpconv:    otelhttpconv.NewHTTPConv(otel.Tracer("httpconv"), otel.Meter("httpconv")),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.serveHTTP(w, r, next)
		})
	}
}

func (m *middleware) serveHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	ctx := m.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// We keep creating the span here, as that is something we want to do before
	// the middleware stack is run
	ctx, span := m.tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.Pattern))
	defer span.End()

	// NewResponseWrapper wraps additional data into the http.ResponseWriter,
	// such as duration, status code and quantity of bytes read.
	rww := otelhttpconv.NewResponseWrapper(w)
	next.ServeHTTP(rww, r.WithContext(ctx))

	// RecordMetrics emits the proper semantic convention metrics
	// With data based on the provided response wrapper
	m.httpconv.RecordMetrics(ctx, rww)

	// RecordSpan emits the proper semantic convention span attributes and events
	// With data based on the provided response wrapper
	// It must not create a new span. It retrieves the current one from the
	// context.
	m.httpconv.RecordSpan(ctx, r, rww)
}
```

### Usage

By default, instrumentations should use the official implementation mentioned
above.
They may provide an option for their users to override the used implementation
with a custom one.

For example, with the `otelhttp` instrumentation for clients:

```golang
otelhttp.NewTransport(http.DefaultTransport, otelhttp.WithHTTPConv(myCustomImplementation{}))
```
