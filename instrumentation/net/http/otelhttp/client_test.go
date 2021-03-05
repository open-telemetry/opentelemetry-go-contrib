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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/oteltest"
)

func TestConvenienceWrappers(t *testing.T) {
	sr := new(oteltest.SpanRecorder)
	provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
	orig := DefaultClient
	DefaultClient = &http.Client{
		Transport: NewTransport(
			http.DefaultTransport,
			WithTracerProvider(provider),
		),
	}
	defer func() { DefaultClient = orig }()

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	res, err := Get(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = Head(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = Post(ctx, ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	form := make(url.Values)
	form.Set("foo", "bar")
	res, err = PostForm(ctx, ts.URL, form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	spans := sr.Completed()
	require.Equal(t, 4, len(spans))
	assert.Equal(t, "GET", spans[0].Name())
	assert.Equal(t, "HEAD", spans[1].Name())
	assert.Equal(t, "POST", spans[2].Name())
	assert.Equal(t, "POST", spans[3].Name())
}
