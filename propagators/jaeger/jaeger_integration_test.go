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

package jaeger_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
)

func TestExtractJaeger(t *testing.T) {
	testGroup := []struct {
		name      string
		testcases []extractTest
	}{
		{
			name:      "valid test case",
			testcases: extractHeaders,
		},
		{
			name:      "invalid test case",
			testcases: invalidExtractHeaders,
		},
	}

	for _, tg := range testGroup {
		propagator := jaeger.Jaeger{}
		props := propagation.New(propagation.WithExtractors(propagator))

		for _, tc := range tg.testcases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}

				ctx := context.Background()
				ctx = propagation.ExtractHTTP(ctx, props, req.Header)
				resSc := trace.RemoteSpanContextFromContext(ctx)
				if diff := cmp.Diff(resSc, tc.expected); diff != "" {
					t.Errorf("%s: %s: -got +want %s", tg.name, tc.name, diff)
				}
			})
		}
	}
}
