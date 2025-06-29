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

package runtimemetrics // import "github.com/open-telemetry/opentelemetry-go-contrib/instrumentation/runtimemetrics"

import (
	"context"
	"fmt"
	"runtime/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// namePrefix is prefixed onto OTel instrument names.
const namePrefix = "process.runtime.go"

// LibraryName is the value of instrumentation.Library.Name.
const LibraryName = "otel-go-contrib/runtimemetrics"

// config contains optional settings for reporting runtime metrics.
type config struct {
	// MeterProvider sets the metric.MeterProvider.  If nil, the global
	// Provider will be used.
	MeterProvider metric.MeterProvider
}

// Option supports configuring optional settings for runtime metrics.
type Option interface {
	apply(*config)
}

// WithMeterProvider sets the Metric implementation to use for
// reporting.  If this option is not used, the global metric.MeterProvider
// will be used.  `provider` must be non-nil.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return metricProviderOption{provider}
}

type metricProviderOption struct{ metric.MeterProvider }

func (o metricProviderOption) apply(c *config) {
	if o.MeterProvider != nil {
		c.MeterProvider = o.MeterProvider
	}
}

// newConfig computes a config from the supplied Options.
func newConfig(opts ...Option) config {
	c := config{
		MeterProvider: otel.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt.apply(&c)
	}
	return c
}

// Start initializes reporting of runtime metrics using the supplied config.
func Start(opts ...Option) error {
	c := newConfig(opts...)
	if c.MeterProvider == nil {
		c.MeterProvider = otel.GetMeterProvider()
	}
	meter := c.MeterProvider.Meter(
		LibraryName,
	)

	r := newBuiltinRuntime(meter, metrics.All, metrics.Read)
	return r.register(expectRuntimeMetrics())
}

// allFunc is the function signature of metrics.All()
type allFunc = func() []metrics.Description

// readFunc is the function signature of metrics.Read()
type readFunc = func([]metrics.Sample)

// builtinRuntime instruments all supported kinds of runtime/metrics.
type builtinRuntime struct {
	meter    metric.Meter
	allFunc  allFunc
	readFunc readFunc
}

func newBuiltinRuntime(meter metric.Meter, af allFunc, rf readFunc) *builtinRuntime {
	return &builtinRuntime{
		meter:    meter,
		allFunc:  af,
		readFunc: rf,
	}
}

// register parses each name and registers metric instruments for all
// the recognized instruments.
func (r *builtinRuntime) register(desc *builtinDescriptor) error {
	all := r.allFunc()

	var instruments []metric.Observable
	var samples []metrics.Sample
	var instAttrs [][]metric.ObserveOption

	for _, m := range all {
		// each should match one
		mname, munit, pattern, attrs, kind, err := desc.findMatch(m.Name)
		if err != nil {
			// skip unrecognized metric names
			otel.Handle(fmt.Errorf("unrecognized runtime/metrics name: %s", m.Name))
			continue
		}
		if kind == builtinSkip {
			// skip e.g., totalized metrics
			continue
		}

		if kind == builtinHistogram {
			// skip unsupported data types
			if m.Kind != metrics.KindFloat64Histogram {
				otel.Handle(fmt.Errorf("expected histogram runtime/metrics: %s", mname))
			}
			continue
		}

		description := fmt.Sprintf("%s from runtime/metrics", pattern)

		unitOpt := metric.WithUnit(munit)
		descOpt := metric.WithDescription(description)

		var inst metric.Observable
		switch kind {
		case builtinCounter:
			switch m.Kind {
			case metrics.KindUint64:
				// e.g., alloc bytes
				inst, err = r.meter.Int64ObservableCounter(mname, unitOpt, descOpt)
			case metrics.KindFloat64:
				// e.g., cpu time (1.20)
				inst, err = r.meter.Float64ObservableCounter(mname, unitOpt, descOpt)
			}
		case builtinUpDownCounter:
			switch m.Kind {
			case metrics.KindUint64:
				// e.g., memory size
				inst, err = r.meter.Int64ObservableUpDownCounter(mname, unitOpt, descOpt)
			case metrics.KindFloat64:
				// not used through 1.20
				inst, err = r.meter.Float64ObservableUpDownCounter(mname, unitOpt, descOpt)
			}
		case builtinGauge:
			switch m.Kind {
			case metrics.KindUint64:
				inst, err = r.meter.Int64ObservableGauge(mname, unitOpt, descOpt)
			case metrics.KindFloat64:
				// not used through 1.20
				inst, err = r.meter.Float64ObservableGauge(mname, unitOpt, descOpt)
			}
		}
		if err != nil {
			return err
		}
		if inst == nil {
			otel.Handle(fmt.Errorf("unexpected runtime/metrics %v: %s", kind, mname))
			continue
		}

		samp := metrics.Sample{
			Name: m.Name,
		}
		samples = append(samples, samp)
		instruments = append(instruments, inst)
		instAttrs = append(instAttrs, []metric.ObserveOption{
			metric.WithAttributes(attrs...),
		})
	}

	if _, err := r.meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		r.readFunc(samples)

		for idx, samp := range samples {
			switch samp.Value.Kind() {
			case metrics.KindUint64:
				obs.ObserveInt64(instruments[idx].(metric.Int64Observable), int64(samp.Value.Uint64()), instAttrs[idx]...)
			case metrics.KindFloat64:
				obs.ObserveFloat64(instruments[idx].(metric.Float64Observable), samp.Value.Float64(), instAttrs[idx]...)
			default:
				// KindFloat64Histogram (unsupported in OTel) and KindBad
				// (unsupported by runtime/metrics).  Neither should happen
				// if runtime/metrics and the code above are working correctly.
				return fmt.Errorf("invalid runtime/metrics value kind: %v", samp.Value.Kind())
			}
		}
		return nil
	}, instruments...); err != nil {
		return err
	}
	return nil
}
