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

package internal // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/internal"

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/stretchr/testify/assert"
)

func TestParseFullMethod(t *testing.T) {
	cases := []struct {
		input         string
		expectedName  string
		expectedAttrs []attribute.KeyValue
	}{
		{
			input:        "no_slash/method",
			expectedName: "no_slash/method",
		},
		{
			input:        "/slash_but_no_second_slash",
			expectedName: "slash_but_no_second_slash",
		},
		{
			input:        "/service_only/",
			expectedName: "service_only/",
			expectedAttrs: []attribute.KeyValue{
				semconv.RPCService("service_only"),
			},
		},
		{
			input:        "//method_only",
			expectedName: "/method_only",
			expectedAttrs: []attribute.KeyValue{
				semconv.RPCMethod("method_only"),
			},
		},
		{
			input:        "/service/method",
			expectedName: "service/method",
			expectedAttrs: []attribute.KeyValue{
				semconv.RPCService("service"),
				semconv.RPCMethod("method"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			name, attrs := ParseFullMethod(tc.input)
			assert.Equal(t, tc.expectedName, name)
			assert.Equal(t, tc.expectedAttrs, attrs)
		})
	}
}
