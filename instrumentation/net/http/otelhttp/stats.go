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

package otelhttp

import (
	"context"
	"go.opentelemetry.io/otel/unit"
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
	valueRecorders map[string]metric.Float64ValueRecorder
}

type tracker struct {
	ctx        context.Context
	start      time.Time
	body       io.ReadCloser
	statusCode int
	endOnce    sync.Once
	labels     *label.Set

	valueRecorders map[string]metric.Float64ValueRecorder
}

// The following tags are applied to stats recorded by this package. Host, Path
// and Method are applied to all measures. StatusCode is not applied to
// ClientRequestCount or ServerRequestCount, since it is recorded before the status is known.
var (
	// Method is the HTTP method of the request, capitalized (GET, POST, etc.).
	Method = label.Key("http.method")
	// Host is the value of the HTTP Host header.
	//
	// The value of this tag can be controlled by the HTTP client, so you need
	// to watch out for potentially generating high-cardinality labels in your
	// metrics backend if you use this tag in views.
	Host = label.Key("http.host")

	// Scheme is the URI scheme identifying the used protocol: "http" or "https"
	Scheme = label.Key("http.scheme")

	// StatusCode is the numeric HTTP response status code,
	// or "error" if a transport error occurred and no status code was read.
	StatusCode = label.Key("http.status")

	// Path is the URL path (not including query string) in the request.
	//
	// The value of this tag can be controlled by the HTTP client, so you need
	// to watch out for potentially generating high-cardinality labels in your
	// metrics backend if you use this tag in views.
	Path = label.Key("http.path")
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
	// KeyClientScheme is the URI scheme identifying the used protocol: "http" or "https"
	KeyClientScheme = label.Key("http_client_host")
)

func (trans *statTransport) applyConfig(c *config) {
	trans.base.applyConfig(c)

	trans.meter = c.Meter
	trans.createMeasures()
}

// RoundTrip implements http.RoundTripper, delegating to Base and recording stats for the request.
func (trans *statTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	labels := label.NewSet(
		// conform open-telemetry label definition
		Method.String(req.Method),
		Host.String(req.Host),
		Scheme.String(req.URL.Scheme),
		Path.String(req.URL.Path),
		// for prometheus
		KeyClientMethod.String(req.Method),
		KeyClientHost.String(req.Host),
		KeyClientScheme.String(req.URL.Scheme),
		KeyClientPath.String(req.URL.Path),
	)

	ctx := req.Context()
	track := &tracker{
		start:          time.Now(),
		ctx:            ctx,
		valueRecorders: trans.valueRecorders,
		labels:         &labels,
	}

	// Perform request.
	resp, err := trans.base.RoundTrip(req)
	if err != nil {
		track.statusCode = http.StatusInternalServerError
		track.end()
	} else {
		track.statusCode = resp.StatusCode
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
	trans.valueRecorders = make(map[string]metric.Float64ValueRecorder)

	requestDurationMeasure, err := trans.meter.NewFloat64ValueRecorder(
		ClientRequestDuration,
		metric.WithDescription("measure the duration of the outbound HTTP request"),
		metric.WithUnit(unit.Milliseconds),
	)
	handleErr(err)

	trans.valueRecorders[ClientRequestDuration] = requestDurationMeasure
}

var _ io.ReadCloser = (*tracker)(nil)

func (t *tracker) end() {
	t.endOnce.Do(func() {
		latencyMs := float64(time.Since(t.start)) / float64(time.Millisecond)
		labels := label.NewSet(
			append(t.labels.ToSlice(),
				StatusCode.Int(t.statusCode),
				KeyClientStatus.Int(t.statusCode),
			)...,
		)
		ls := labels.ToSlice()

		t.valueRecorders[ClientRequestDuration].Record(t.ctx, latencyMs, ls...)
	})
}

func (t *tracker) Read(b []byte) (int, error) {
	n, err := t.body.Read(b)
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
