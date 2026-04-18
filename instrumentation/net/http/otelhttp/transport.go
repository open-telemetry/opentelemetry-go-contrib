// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/request"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv"
)

// Transport implements the http.RoundTripper interface and wraps
// outbound HTTP(S) requests with a span and enriches it with metrics.
type Transport struct {
	rt http.RoundTripper

	tracer             trace.Tracer
	propagators        propagation.TextMapPropagator
	spanStartOptions   []trace.SpanStartOption
	filters            []Filter
	spanNameFormatter  func(string, *http.Request) string
	clientTrace        func(context.Context) *httptrace.ClientTrace
	metricAttributesFn func(*http.Request) []attribute.KeyValue

	semconv semconv.HTTPClient
}

var _ http.RoundTripper = &Transport{}

// NewTransport wraps the provided http.RoundTripper with one that
// starts a span, injects the span context into the outbound request headers,
// and enriches it with metrics.
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

	return &t
}

func (t *Transport) applyConfig(c *config) {
	t.tracer = c.Tracer
	t.propagators = c.Propagators
	t.spanStartOptions = c.SpanStartOptions
	t.filters = c.Filters
	t.spanNameFormatter = c.SpanNameFormatter
	t.clientTrace = c.ClientTrace
	t.semconv = semconv.NewHTTPClient(c.Meter)
	t.metricAttributesFn = c.MetricAttributesFn
}

func defaultTransportFormatter(_ string, r *http.Request) string {
	return "HTTP " + r.Method
}

type requestBodyTracker struct {
	*request.BodyWrapper

	onClose   func()
	closeOnce sync.Once
	closed    atomic.Bool
}

func newRequestBodyTracker(body io.ReadCloser, onClose func()) *requestBodyTracker {
	return &requestBodyTracker{
		BodyWrapper: request.NewBodyWrapper(body, func(int64) {}),
		onClose:     onClose,
	}
}

func (w *requestBodyTracker) Close() error {
	err := w.BodyWrapper.Close()
	w.closed.Store(true)
	w.closeOnce.Do(func() {
		w.onClose()
	})
	return err
}

func (w *requestBodyTracker) Closed() bool {
	return w.closed.Load()
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

	ctx, span := tracer.Start(r.Context(), t.spanNameFormatter("", r), t.spanStartOptions...)

	if t.clientTrace != nil {
		ctx = httptrace.WithClientTrace(ctx, t.clientTrace(ctx))
	}

	labeler, found := LabelerFromContext(ctx)
	if !found {
		ctx = ContextWithLabeler(ctx, labeler)
	}

	r = r.Clone(ctx) // According to RoundTripper spec, we shouldn't modify the origin request.

	var (
		lastBodyMu      sync.Mutex
		lastTrackedBody *requestBodyTracker
		recordMetrics   atomic.Pointer[func()]
	)
	emptyFn := func() {}
	recordMetrics.Store(&emptyFn)
	setLastTrackedBody := func(body *requestBodyTracker) {
		lastBodyMu.Lock()
		lastTrackedBody = body
		lastBodyMu.Unlock()
	}
	currentTrackedBody := func() *requestBodyTracker {
		lastBodyMu.Lock()
		defer lastBodyMu.Unlock()
		return lastTrackedBody
	}
	maybeWrapBody := func(body io.ReadCloser) io.ReadCloser {
		if body == nil || body == http.NoBody {
			return body
		}
		var trackedBody *requestBodyTracker
		trackedBody = newRequestBodyTracker(body, func() {
			if currentTrackedBody() == trackedBody {
				(*recordMetrics.Load())()
			}
		})
		setLastTrackedBody(trackedBody)
		return trackedBody
	}
	r.Body = maybeWrapBody(r.Body)
	if r.GetBody != nil {
		originalGetBody := r.GetBody
		r.GetBody = func() (io.ReadCloser, error) {
			b, err := originalGetBody()
			if err != nil {
				setLastTrackedBody(nil) // The underlying transport will fail to make a retry request, hence, record no data.
				return nil, err
			}
			return maybeWrapBody(b), nil
		}
	}

	span.SetAttributes(t.semconv.RequestTraceAttrs(r)...)
	t.propagators.Inject(ctx, propagation.HeaderCarrier(r.Header))

	res, err := t.rt.RoundTrip(r)

	requestDuration := time.Since(requestStartTime)
	statusCode := 0
	if err == nil {
		statusCode = res.StatusCode
	}
	metricOptions := t.semconv.MetricOptions(semconv.MetricAttributes{
		Req:                  r,
		StatusCode:           statusCode,
		Err:                  err,
		AdditionalAttributes: append(labeler.Get(), t.metricAttributesFromRequest(r)...),
	})

	// Delay metric recording until the response body is finalized. The
	// transport can continue reading the request body after RoundTrip returns.
	var recordMetricsOnce sync.Once
	realRecordMetrics := func() {
		var requestSize int64
		if lastTrackedBody := currentTrackedBody(); lastTrackedBody != nil {
			requestSize = lastTrackedBody.BytesRead()
		}
		recordMetricsOnce.Do(func() {
			t.semconv.RecordMetrics(
				ctx,
				semconv.MetricData{
					RequestSize:     requestSize,
					RequestDuration: requestDuration,
				},
				metricOptions,
			)
		})
	}
	recordMetrics.Store(&realRecordMetrics)

	if err != nil {
		recordMetrics()
		span.SetAttributes(otelsemconv.ErrorType(err))
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return res, err
	}

	readRecordFunc := func(int64) {
		lastTrackedBody := currentTrackedBody()
		if lastTrackedBody == nil || lastTrackedBody.Closed() {
			recordMetrics()
		}
	}
	res.Body = newWrappedBody(span, readRecordFunc, res.Body)
	// traces
	span.SetAttributes(t.semconv.ResponseTraceAttrs(res)...)
	span.SetStatus(t.semconv.Status(res.StatusCode))

	return res, nil
}

func (t *Transport) metricAttributesFromRequest(r *http.Request) []attribute.KeyValue {
	var attributeForRequest []attribute.KeyValue
	if t.metricAttributesFn != nil {
		attributeForRequest = t.metricAttributesFn(r)
	}
	return attributeForRequest
}

// newWrappedBody returns a new and appropriately scoped *wrappedBody as an
// io.ReadCloser. If the passed body implements io.Writer, the returned value
// will implement io.ReadWriteCloser.
func newWrappedBody(span trace.Span, record func(n int64), body io.ReadCloser) io.ReadCloser {
	// The successful protocol switch responses will have a body that
	// implement an io.ReadWriteCloser. Ensure this interface type continues
	// to be satisfied if that is the case.
	if _, ok := body.(io.ReadWriteCloser); ok {
		return &wrappedBody{span: span, record: record, body: body}
	}

	// Remove the implementation of the io.ReadWriteCloser and only implement
	// the io.ReadCloser.
	return struct{ io.ReadCloser }{&wrappedBody{span: span, record: record, body: body}}
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
	span   trace.Span
	record func(n int64)
	body   io.ReadCloser
	read   atomic.Int64

	closeOnce    sync.Once
	closeErr     error
	finalizeOnce sync.Once
}

var _ io.ReadWriteCloser = &wrappedBody{}

func (wb *wrappedBody) Write(p []byte) (int, error) {
	// This will not panic given the guard in newWrappedBody.
	n, err := wb.body.(io.Writer).Write(p)
	if err != nil {
		wb.span.SetAttributes(otelsemconv.ErrorType(err))
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Read(b []byte) (int, error) {
	n, err := wb.body.Read(b)
	// Record the number of bytes read
	wb.read.Add(int64(n))

	switch err {
	case nil:
		// nothing to do here but fall through to the return
	case io.EOF:
		wb.recordMetricsOnce()
	default:
		wb.span.SetAttributes(otelsemconv.ErrorType(err))
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) closeBody() error {
	wb.closeOnce.Do(func() {
		if wb.body != nil {
			wb.closeErr = wb.body.Close()
		}
	})

	return wb.closeErr
}

// recordMetricsOnce ensures the final number of bytes read is recorded once.
func (wb *wrappedBody) recordMetricsOnce() {
	wb.finalizeOnce.Do(func() {
		wb.record(wb.read.Load())
		wb.span.End()
	})
}

func (wb *wrappedBody) Close() error {
	err := wb.closeBody()
	wb.recordMetricsOnce()
	return err
}
