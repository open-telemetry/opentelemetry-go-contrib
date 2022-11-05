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

package otelsarama

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestNewConfig(t *testing.T) {
	tp := trace.NewNoopTracerProvider()
	mp := metric.NewNoopMeterProvider()

	prop := propagation.NewCompositeTextMapPropagator()

	testCases := []struct {
		name     string
		opts     []Option
		expected config
	}{
		{
			name: "with both providers",
			opts: []Option{
				WithTracerProvider(tp),
				WithMeterProvider(mp),
			},
			expected: config{
				TracerProvider: tp,
				MeterProvider:  mp,
				Tracer:         tp.Tracer(defaultObservabilityName, trace.WithInstrumentationVersion(SemVersion())),
				Meter:          mp.Meter(defaultObservabilityName, metric.WithInstrumentationVersion(SemVersion())),
				Propagators:    otel.GetTextMapPropagator(),
			},
		},
		{
			name: "with both empty providers",
			opts: []Option{
				WithTracerProvider(nil),
				WithMeterProvider(nil),
			},
			expected: config{
				TracerProvider: otel.GetTracerProvider(),
				MeterProvider:  global.MeterProvider(),
				Tracer:         otel.GetTracerProvider().Tracer(defaultObservabilityName, trace.WithInstrumentationVersion(SemVersion())),
				Meter:          global.MeterProvider().Meter(defaultObservabilityName, metric.WithInstrumentationVersion(SemVersion())),
				Propagators:    otel.GetTextMapPropagator(),
			},
		},

		{
			name: "with propagators",
			opts: []Option{
				WithPropagators(prop),
			},
			expected: config{
				TracerProvider: otel.GetTracerProvider(),
				MeterProvider:  global.MeterProvider(),
				Tracer:         otel.GetTracerProvider().Tracer(defaultObservabilityName, trace.WithInstrumentationVersion(SemVersion())),
				Meter:          global.MeterProvider().Meter(defaultObservabilityName, metric.WithInstrumentationVersion(SemVersion())),
				Propagators:    prop,
			},
		},
		{
			name: "with empty propagators",
			opts: []Option{
				WithPropagators(nil),
			},
			expected: config{
				TracerProvider: otel.GetTracerProvider(),
				MeterProvider:  global.MeterProvider(),
				Tracer:         otel.GetTracerProvider().Tracer(defaultObservabilityName, trace.WithInstrumentationVersion(SemVersion())),
				Meter:          global.MeterProvider().Meter(defaultObservabilityName, metric.WithInstrumentationVersion(SemVersion())),
				Propagators:    otel.GetTextMapPropagator(),
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
