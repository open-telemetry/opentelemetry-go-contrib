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

package otelhttp_test

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func assertActiveMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, actual metricdata.Aggregation, opts ...metricdatatest.Option) bool {
	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)

	if !assert.NoError(t, err, "failed to read the metric reader") {
		return false
	}

	if !assert.Len(t, rm.ScopeMetrics, 1, "too many metrics") {
		return false
	}

	metrics := rm.ScopeMetrics[0].Metrics
	m := metrics[slices.IndexFunc(metrics, func(m metricdata.Metrics) bool {
		return m.Name == otelhttp.ActiveRequests
	})]
	return metricdatatest.AssertAggregationsEqual(t, actual, m.Data, opts...)
}

func activeMetric(val int64) metricdata.Sum[int64] {
	return metricdata.Sum[int64]{
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: false,
		DataPoints: []metricdata.DataPoint[int64]{
			{
				Value: val,
				Attributes: attribute.NewSet(
					semconv.HTTPMethod("GET"),
					semconv.HTTPScheme("http"),
				),
			},
		},
	}
}

func TestActiveMetrics(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	reader := metric.NewManualReader()
	var g sync.WaitGroup
	var ag sync.WaitGroup
	var l sync.Mutex

	handler := otelhttp.NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		g.Done()
		l.Lock()
		defer l.Unlock()
		ag.Done()
	}), "test", otelhttp.WithMeterProvider(metric.NewMeterProvider(metric.WithReader(reader))))

	g.Add(5)
	ag.Add(5)
	l.Lock()

	for i := 0; i < 5; i++ {
		go handler.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("GET", "/foo/bar", nil),
		)
	}

	g.Wait()

	assertActiveMetric(t, ctx, reader, activeMetric(5), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	g.Add(5)
	ag.Add(5)
	for i := 0; i < 5; i++ {
		go handler.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("GET", "/foo/bar", nil),
		)
	}
	g.Wait()

	assertActiveMetric(t, ctx, reader, activeMetric(10), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	l.Unlock()
	ag.Wait()

	assertActiveMetric(t, ctx, reader, activeMetric(0), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		name               string
		handler            func(*testing.T) http.Handler
		requestBody        io.Reader
		expectedStatusCode int
	}{
		{
			name: "implements flusher",
			handler: func(t *testing.T) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Implements(t, (*http.Flusher)(nil), w)
					}), "test_handler",
				)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds",
			handler: func(t *testing.T) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.NotNil(t, r.Body)

						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						assert.Equal(t, "hello world", string(b))
					}), "test_handler",
				)
			},
			requestBody:        strings.NewReader("hello world"),
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds with a nil body",
			handler: func(t *testing.T) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Nil(t, r.Body)
					}), "test_handler",
				)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds with an http.NoBody",
			handler: func(t *testing.T) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, http.NoBody, r.Body)
					}), "test_handler",
				)
			},
			requestBody:        http.NoBody,
			expectedStatusCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "http://localhost/", tc.requestBody)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			tc.handler(t).ServeHTTP(rr, r)
			assert.Equal(t, tc.expectedStatusCode, rr.Result().StatusCode)
		})
	}
}
