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
		name        string
		serviceName string
		opts        []Option
		expected    config
	}{
		{
			name:        "set service name",
			serviceName: serviceName,
			expected: config{
				ServiceName: serviceName,
				Tracer:      global.Tracer(defaultTracerName),
				Propagators: global.Propagators(),
			},
		},
		{
			name:        "with tracer",
			serviceName: serviceName,
			opts: []Option{
				WithTracer(global.Tracer("new")),
			},
			expected: config{
				ServiceName: serviceName,
				Tracer:      global.Tracer("new"),
				Propagators: global.Propagators(),
			},
		},
		{
			name:        "with propagators",
			serviceName: serviceName,
			opts: []Option{
				WithPropagators(nil),
			},
			expected: config{
				ServiceName: serviceName,
				Tracer:      global.Tracer(defaultTracerName),
				Propagators: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := newConfig(tc.serviceName, tc.opts...)
			assert.Equal(t, tc.expected, result)
		})
	}
}
