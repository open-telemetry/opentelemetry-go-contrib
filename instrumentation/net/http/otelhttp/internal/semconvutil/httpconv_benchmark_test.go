// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/semconvutil/httpconv_test.go.tmpl

// Copyright The OpenTelemetry Authors
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

package semconvutil

import (
	"net/http"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

var benchHTTPServerRequestResults []attribute.KeyValue

func BenchmarkHTTPServerRequest(b *testing.B) {
	// Request was generated from TestHTTPServerRequest request.
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Path: "/",
		},
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"User-Agent":      []string{"Go-http-client/1.1"},
			"Accept-Encoding": []string{"gzip"},
		},
		Body:       http.NoBody,
		Host:       "127.0.0.1:39093",
		RemoteAddr: "127.0.0.1:38738",
		RequestURI: "/",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchHTTPServerRequestResults = HTTPServerRequest("", req)
	}
}

var benchHTTPServerRequestMetricsResults []attribute.KeyValue

func BenchmarkHTTPServerRequestMetrics(b *testing.B) {
	// Request was generated from TestHTTPServerRequestMetrics request.
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Path: "/",
		},
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"User-Agent":      []string{"Go-http-client/1.1"},
			"Accept-Encoding": []string{"gzip"},
		},
		Body:       http.NoBody,
		Host:       "127.0.0.1:39093",
		RemoteAddr: "127.0.0.1:38738",
		RequestURI: "/",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchHTTPServerRequestMetricsResults = HTTPServerRequestMetrics("", req)
	}
}
