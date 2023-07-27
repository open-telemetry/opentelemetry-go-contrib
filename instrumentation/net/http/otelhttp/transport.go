// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconvutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Transport implements the http.RoundTripper interface and wraps
// outbound HTTP(S) requests with a span.
type Transport struct {
	rt http.RoundTripper

	tracer                trace.Tracer
	meter                 metric.Meter
	propagators           propagation.TextMapPropagator
	spanStartOptions      []trace.SpanStartOption
	readEvent             bool
	filters               []Filter
	spanNameFormatter     func(string, *http.Request) string
	clientTrace           func(context.Context) *httptrace.ClientTrace
	getRequestAttributes  func(*http.Request) []attribute.KeyValue
	getResponseAttributes func(response *http.Response) []attribute.KeyValue
	counters              map[string]metric.Int64Counter
	valueRecorders        map[string]metric.Float64Histogram
}

var _ http.RoundTripper = &Transport{}

// NewTransport wraps the provided http.RoundTripper with one that
// starts a span and injects the span context into the outbound request headers.
//
// If the provided http.RoundTripper is nil, http.DefaultTransport will be used
// as the base http.RoundTripper.
func NewTransport(base http.RoundTripper, opts ...Option) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}

	t := Transport{
		rt: base,
	}

	defaultOpts := []Option{
		WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)),
		WithSpanNameFormatter(defaultTransportFormatter),
	}

	c := newConfig(append(defaultOpts, opts...)...)
	t.applyConfig(c)
	t.createMeasures()

	return &t
}

func (t *Transport) applyConfig(c *config) {
	t.tracer = c.Tracer
	t.meter = c.Meter
	t.propagators = c.Propagators
	t.spanStartOptions = c.SpanStartOptions
	t.readEvent = c.ReadEvent
	t.filters = c.Filters
	t.spanNameFormatter = c.SpanNameFormatter
	t.clientTrace = c.ClientTrace
	t.getRequestAttributes = c.GetRequestAttributes
	t.getResponseAttributes = c.GetResponseAttributes
}

func defaultTransportFormatter(_ string, r *http.Request) string {
	return "HTTP " + r.Method
}

func (t *Transport) createMeasures() {
	t.counters = make(map[string]metric.Int64Counter)
	t.valueRecorders = make(map[string]metric.Float64Histogram)

	requestBytesCounter, err := t.meter.Int64Counter(ClientRequestContentLength)
	handleErr(err)

	responseBytesCounter, err := t.meter.Int64Counter(ClientResponseContentLength)
	handleErr(err)

	clientLatencyMeasure, err := t.meter.Float64Histogram(ClientLatency)
	handleErr(err)

	t.counters[ClientRequestContentLength] = requestBytesCounter
	t.counters[ClientResponseContentLength] = responseBytesCounter
	t.valueRecorders[ClientLatency] = clientLatencyMeasure
}

// RoundTrip creates a Span and propagates its context via the provided request's headers
// before handing the request to the configured base RoundTripper. The created span will
// end when the response body is closed or when a read from the body returns io.EOF.
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	requestStartTime := time.Now()
	for _, f := range t.filters {
		if !f(r) {
			// Simply pass through to the base RoundTripper if a filter rejects the request
			return t.rt.RoundTrip(r)
		}
	}

	tracer := t.tracer

	if tracer == nil {
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			tracer = newTracer(span.TracerProvider())
		} else {
			tracer = newTracer(otel.GetTracerProvider())
		}
	}

	opts := append([]trace.SpanStartOption{}, t.spanStartOptions...) // start with the configured options

	ctx, span := tracer.Start(r.Context(), t.spanNameFormatter("", r), opts...)

	if t.clientTrace != nil {
		ctx = httptrace.WithClientTrace(ctx, t.clientTrace(ctx))
	}

	readRecordFunc := func(int64) {}
	if t.readEvent {
		readRecordFunc = func(n int64) {
			span.AddEvent("read", trace.WithAttributes(ReadBytesKey.Int64(n)))
		}
	}

	var bw bodyWrapper
	// if request body is nil or NoBody, we don't want to mutate the body as it
	// will affect the identity of it in an unforeseeable way because we assert
	// ReadCloser fulfills a certain interface, and it is indeed nil or NoBody.
	if r.Body != nil && r.Body != http.NoBody {
		bw.ReadCloser = r.Body
		bw.record = readRecordFunc
		r.Body = &bw
	}

	r = r.Clone(ctx) // According to RoundTripper spec, we shouldn't modify the origin request.
	span.SetAttributes(semconvutil.HTTPClientRequest(r)...)
	if t.getRequestAttributes != nil {
		span.SetAttributes(t.getRequestAttributes(r)...)
	}
	t.propagators.Inject(ctx, propagation.HeaderCarrier(r.Header))

	res, err := t.rt.RoundTrip(r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
	} else {
		span.SetAttributes(httpconv.ClientResponse(res)...)
		if t.getResponseAttributes != nil {
			span.SetAttributes(t.getResponseAttributes(res)...)
		}
		span.SetStatus(httpconv.ClientStatus(res.StatusCode))
		res.Body = newWrappedBody(span, res.Body)
	}

	// Add metrics
	attributes := httpconv.ClientRequest(r)
	if t.getRequestAttributes != nil {
		attributes = append(attributes, t.getRequestAttributes(r)...)
	}
	if err == nil {
		attributes = append(attributes, httpconv.ClientResponse(res)...)
		if t.getResponseAttributes != nil {
			attributes = append(attributes, t.getResponseAttributes(res)...)
		}
	}
	o := metric.WithAttributes(attributes...)
	t.counters[ClientRequestContentLength].Add(ctx, bw.read.Load(), o)
	if err == nil {
		t.counters[ClientResponseContentLength].Add(ctx, res.ContentLength, o)
	}

	// Use floating point division here for higher precision (instead of Millisecond method).
	elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)
	t.valueRecorders[ClientLatency].Record(ctx, elapsedTime, o)

  span.SetAttributes(semconvutil.HTTPClientResponse(res)...)
	span.SetStatus(semconvutil.HTTPClientStatus(res.StatusCode))
	res.Body = newWrappedBody(span, res.Body)

	return res, err
}

// newWrappedBody returns a new and appropriately scoped *wrappedBody as an
// io.ReadCloser. If the passed body implements io.Writer, the returned value
// will implement io.ReadWriteCloser.
func newWrappedBody(span trace.Span, body io.ReadCloser) io.ReadCloser {
	// The successful protocol switch responses will have a body that
	// implement an io.ReadWriteCloser. Ensure this interface type continues
	// to be satisfied if that is the case.
	if _, ok := body.(io.ReadWriteCloser); ok {
		return &wrappedBody{span: span, body: body}
	}

	// Remove the implementation of the io.ReadWriteCloser and only implement
	// the io.ReadCloser.
	return struct{ io.ReadCloser }{&wrappedBody{span: span, body: body}}
}

// wrappedBody is the response body type returned by the transport
// instrumentation to complete a span. Errors encountered when using the
// response body are recorded in span tracking the response.
//
// The span tracking the response is ended when this body is closed.
//
// If the response body implements the io.Writer interface (i.e. for
// successful protocol switches), the wrapped body also will.
type wrappedBody struct {
	span trace.Span
	body io.ReadCloser
}

var _ io.ReadWriteCloser = &wrappedBody{}

func (wb *wrappedBody) Write(p []byte) (int, error) {
	// This will not panic given the guard in newWrappedBody.
	n, err := wb.body.(io.Writer).Write(p)
	if err != nil {
		wb.span.RecordError(err)
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Read(b []byte) (int, error) {
	n, err := wb.body.Read(b)

	switch err {
	case nil:
		// nothing to do here but fall through to the return
	case io.EOF:
		wb.span.End()
	default:
		wb.span.RecordError(err)
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Close() error {
	wb.span.End()
	if wb.body != nil {
		return wb.body.Close()
	}
	return nil
}
