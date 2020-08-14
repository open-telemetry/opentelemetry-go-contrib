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

package sarama

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/api/global"
)

func TestNewConfig(t *testing.T) {
	testCases := []struct {
		name     string
		opts     []Option
		expected config
	}{
		{
			name: "with provider",
			opts: []Option{
				WithTraceProvider(global.TraceProvider()),
			},
			expected: config{
				TraceProvider: global.TraceProvider(),
				Tracer:        global.TraceProvider().Tracer(defaultTracerName),
				Propagators:   global.Propagators(),
			},
		},
		{
			name: "with propagators",
			opts: []Option{
				WithPropagators(nil),
			},
			expected: config{
				TraceProvider: global.TraceProvider(),
				Tracer:        global.TraceProvider().Tracer(defaultTracerName),
				Propagators:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := newConfig(tc.opts...)
			assert.Equal(t, tc.expected, result)
		})
	}
}
