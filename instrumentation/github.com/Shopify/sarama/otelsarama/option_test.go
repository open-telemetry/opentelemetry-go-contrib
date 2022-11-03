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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// We need a fake tracer provider to ensure the one passed in options is the one used afterwards.
// In order to avoid adding the SDK as a dependency, we use this mock.
type fakeTracerProvider struct{}

func (fakeTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return fakeTracer{
		name: name,
	}
}

type fakeTracer struct {
	name string
}

func (fakeTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, nil
}

type fakeMeterProvider struct{}

func (f fakeMeterProvider) Meter(instrumentationName string, opts ...metric.MeterOption) metric.Meter {
	return fakeMeter{
		instrumentationName: instrumentationName,
	}
}

type fakeMeter struct {
	instrumentationName string
}

func (f fakeMeter) AsyncInt64() asyncint64.InstrumentProvider {
	return nil
}
func (fakeMeter) AsyncFloat64() asyncfloat64.InstrumentProvider {
	return nil
}
func (fakeMeter) RegisterCallback(insts []instrument.Asynchronous, function func(context.Context)) error {
	return nil
}
func (fakeMeter) SyncInt64() syncint64.InstrumentProvider {
	return nil
}
func (fakeMeter) SyncFloat64() syncfloat64.InstrumentProvider {
	return nil
}

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
