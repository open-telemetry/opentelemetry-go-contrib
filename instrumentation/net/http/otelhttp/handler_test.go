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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestResponseWriterImplementsFlusher(t *testing.T) {
	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Implements(t, (*http.Flusher)(nil), w)
		}), "test_handler",
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)

	h.ServeHTTP(httptest.NewRecorder(), r)
}

// This use case is important as we make sure the body isn't mutated
// when it is nil. This is a common use case for tests where the request
// is directly passed to the handler.
func TestHandlerReadingNilBodySuccess(t *testing.T) {
	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				_, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
			}
		}), "test_handler",
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	assert.Equal(t, 200, rr.Result().StatusCode)
}

// This use case is important as we make sure the body isn't mutated
// when it is NoBody.
func TestHandlerReadingNoBodySuccess(t *testing.T) {
	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != http.NoBody {
				_, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
			}
		}), "test_handler",
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	assert.Equal(t, 200, rr.Result().StatusCode)
}
