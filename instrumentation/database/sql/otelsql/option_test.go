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

package otelsql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/oteltest"
)

func TestOptions(t *testing.T) {
	tracerProvider := oteltest.NewTracerProvider()

	testCases := []struct {
		name           string
		option         Option
		expectedConfig config
	}{
		{
			name:           "WithTracerProvider",
			option:         WithTracerProvider(tracerProvider),
			expectedConfig: config{TracerProvider: tracerProvider},
		},
		{
			name: "WithAttributes",
			option: WithAttributes(
				attribute.String("foo", "bar"),
				attribute.String("foo2", "bar2"),
			),
			expectedConfig: config{Attributes: []attribute.KeyValue{
				attribute.String("foo", "bar"),
				attribute.String("foo2", "bar2"),
			}},
		},
		{
			name:           "WithSpanNameFormatter",
			option:         WithSpanNameFormatter(&defaultSpanNameFormatter{}),
			expectedConfig: config{SpanNameFormatter: &defaultSpanNameFormatter{}},
		},
		{
			name:           "WithSpanOptions",
			option:         WithSpanOptions(SpanOptions{Ping: true}),
			expectedConfig: config{SpanOptions: SpanOptions{Ping: true}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg config

			tc.option.Apply(&cfg)

			assert.Equal(t, tc.expectedConfig, cfg)
		})
	}
}
