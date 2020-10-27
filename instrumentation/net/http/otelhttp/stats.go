package otelhttp

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
)

type statTransport struct {
	meter          metric.Meter
	base           *Transport
	labels         label.Set
	counters       map[string]metric.Int64Counter
	valueRecorders map[string]metric.Float64ValueRecorder
}

type tracker struct {
	ctx               context.Context
	respSize          int64
	respContentLength int64
	reqSize           int64
	start             time.Time
	body              io.ReadCloser
	statusCode        int
	endOnce           sync.Once
	labels            label.Set

	counters       map[string]metric.Int64Counter
	valueRecorders map[string]metric.Float64ValueRecorder
}

// The following tags are applied to stats recorded by this package. Host, Path
// and Method are applied to all measures. StatusCode is not applied to
// ClientRequestCount or ServerRequestCount, since it is recorded before the status is known.
var (
	// Host is the value of the HTTP Host header.
	//
	// The value of this tag can be controlled by the HTTP client, so you need
	// to watch out for potentially generating high-cardinality labels in your
	// metrics backend if you use this tag in views.
	Host = label.Key("http.host")

	// StatusCode is the numeric HTTP response status code,
	// or "error" if a transport error occurred and no status code was read.
	StatusCode = label.Key("http.status")

	// Path is the URL path (not including query string) in the request.
	//
	// The value of this tag can be controlled by the HTTP client, so you need
	// to watch out for potentially generating high-cardinality labels in your
	// metrics backend if you use this tag in views.
	Path = label.Key("http.path")

	// Method is the HTTP method of the request, capitalized (GET, POST, etc.).
	Method = label.Key("http.method")
)

// Client tag keys.
var (
	// KeyClientMethod is the HTTP method, capitalized (i.e. GET, POST, PUT, DELETE, etc.).
	KeyClientMethod = label.Key("http_client_method")
	// KeyClientPath is the URL path (not including query string).
	KeyClientPath = label.Key("http_client_path")
	// KeyClientStatus is the HTTP status code as an integer (e.g. 200, 404, 500.), or "error" if no response status line was received.
	KeyClientStatus = label.Key("http_client_status")
	// KeyClientHost is the value of the request Host header.
	KeyClientHost = label.Key("http_client_host")
)

func (trans *statTransport) applyConfig(c *config) {
	trans.base.applyConfig(c)

	trans.meter = c.Meter
	trans.createMeasures()
}

// RoundTrip implements http.RoundTripper, delegating to Base and recording stats for the request.
func (trans *statTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	trans.labels = label.NewSet(
		KeyClientHost.String(req.Host),
		Host.String(req.Host),
		KeyClientPath.String(req.URL.Path),
		Path.String(req.URL.Path),
		KeyClientMethod.String(req.Method),
		Method.String(req.Method),
	)

	ctx := req.Context()
	track := &tracker{
		start:          time.Now(),
		ctx:            ctx,
		counters:       trans.counters,
		valueRecorders: trans.valueRecorders,
	}
	if req.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if req.ContentLength > 0 {
		track.reqSize = req.ContentLength
	}
	trans.counters[ClientRequestCount].Add(ctx, 1, trans.labels.ToSlice()...)

	// Perform request.
	resp, err := trans.base.RoundTrip(req)

	if err != nil {
		track.statusCode = http.StatusInternalServerError
		track.end()
	} else {
		track.statusCode = resp.StatusCode
		if req.Method != "HEAD" {
			track.respContentLength = resp.ContentLength
		}
		if resp.Body == nil {
			track.end()
		} else {
			track.body = resp.Body
			resp.Body = wrappedBodyIO(track, resp.Body)
		}
	}
	return resp, err
}

// wrappedBodyIO returns a wrapped version of the original
// Body and only implements the same combination of additional
// interfaces as the original.
func wrappedBodyIO(wrapper io.ReadCloser, body io.ReadCloser) io.ReadCloser {
	wr, i0 := body.(io.Writer)
	switch {
	case !i0:
		return struct {
			io.ReadCloser
		}{wrapper}

	case i0:
		return struct {
			io.ReadCloser
			io.Writer
		}{wrapper, wr}
	default:
		return struct {
			io.ReadCloser
		}{wrapper}
	}
}

func (trans *statTransport) createMeasures() {
	trans.counters = make(map[string]metric.Int64Counter)
	trans.valueRecorders = make(map[string]metric.Float64ValueRecorder)

	clientRequestCountCounter, err := trans.meter.NewInt64Counter(ClientRequestCount)
	handleErr(err)

	requestBytesCounter, err := trans.meter.NewInt64Counter(ClientRequestContentLength)
	handleErr(err)

	responseBytesCounter, err := trans.meter.NewInt64Counter(ClientResponseContentLength)
	handleErr(err)

	serverLatencyMeasure, err := trans.meter.NewFloat64ValueRecorder(ClientRoundTripLatency)
	handleErr(err)

	trans.counters[ClientRequestCount] = clientRequestCountCounter
	trans.counters[ClientRequestContentLength] = requestBytesCounter
	trans.counters[ClientResponseContentLength] = responseBytesCounter
	trans.valueRecorders[ClientRoundTripLatency] = serverLatencyMeasure
}

var _ io.ReadCloser = (*tracker)(nil)

func (t *tracker) end() {
	t.endOnce.Do(func() {
		latencyMs := float64(time.Since(t.start)) / float64(time.Millisecond)
		respSize := t.respSize
		if t.respSize == 0 && t.respContentLength > 0 {
			respSize = t.respContentLength
		}
		labels := label.NewSet(
			append(t.labels.ToSlice(), StatusCode.Int(t.statusCode),
				KeyClientStatus.Int(t.statusCode))...,
		)
		ls := labels.ToSlice()

		t.counters[ClientResponseContentLength].Add(t.ctx, respSize, ls...)
		t.valueRecorders[ClientRoundTripLatency].Record(t.ctx, latencyMs, ls...)
		if t.reqSize >= 0 {
			t.counters[ClientRequestContentLength].Add(t.ctx, respSize, ls...)
		}
	})
}

func (t *tracker) Read(b []byte) (int, error) {
	n, err := t.body.Read(b)
	t.respSize += int64(n)
	switch err {
	case nil:
		return n, nil
	case io.EOF:
		t.end()
	}
	return n, err
}

func (t *tracker) Close() error {
	// Invoking endSpan on Close will help catch the cases
	// in which a read returned a non-nil error, we set the
	// span status but didn't end the span.
	t.end()
	return t.body.Close()
}
