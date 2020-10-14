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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/api/global"

	mocktrace "go.opentelemetry.io/contrib/internal/trace"
)

func TestConvenienceWrappers(t *testing.T) {
	provider, tracer := mocktrace.NewTracerProviderAndTracer(instrumentationName)
	global.SetTracerProvider(provider)

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("traceparent") == "" {
			t.Fatal("Expected traceparent header")
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	context, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	_, err := Get(context, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Head(context, ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Post(context, ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatal(err)
	}

	form := make(url.Values)
	form.Set("foo", "bar")
	_, err = PostForm(context, ts.URL, form)
	if err != nil {
		t.Fatal(err)
	}

}
