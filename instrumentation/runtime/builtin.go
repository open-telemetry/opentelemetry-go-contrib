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

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"context"
	"fmt"
	"runtime/metrics"
	"strings"

	"github.com/hashicorp/go-multierror"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
)

type allFunc = func() []metrics.Description
type readFunc = func([]metrics.Sample)

type builtinRuntime struct {
	meter       metric.Meter
	samples     []metrics.Sample
	instruments []instrument.Asynchronous
}

type int64Observer interface {
	Observe(ctx context.Context, x int64, attrs ...attribute.KeyValue)
}

type float64Observer interface {
	Observe(ctx context.Context, x float64, attrs ...attribute.KeyValue)
}

func newBuiltinRuntime(meter metric.Meter, af allFunc, rf readFunc) *builtinRuntime {
	return &builtinRuntime{
		meter: meter,
	}
}

func (r *builtinRuntime) register() error {
	all := metrics.All()
	counts := map[string]int{}
	toName := func(in string) (string, string) {
		n, statedUnits, _ := strings.Cut(in, ":")
		n = "process.runtime.go" + strings.ReplaceAll(n, "/", ".")
		return n, statedUnits
	}

	for _, m := range all {
		name, _ := toName(m.Name)
		counts[name]++
	}

	var rerr error
	for _, m := range all {
		n, statedUnits := toName(m.Name)
		var u string
		switch statedUnits {
		case "bytes", "seconds":
			// Real units
			u = statedUnits
		default:
			// Pseudo-units
			u = "{" + statedUnits + "}"
		}

		if counts[n] > 1 {
			// This is treated as a special case, we know this happens
			// with "objects" and "bytes" in the standard Go 1.19 runtime.
			switch statedUnits {
			case "objects":
				// In this case, use `.objects` suffix.
				n = n + ".objects"
				u = "{objects}"
			case "bytes":
				// In this case, use no suffix.  In Prometheus this will
				// be appended as a suffix.
			default:
				panic(fmt.Sprint(
					"unrecognized duplicate metrics names, ",
					"attention required: ",
					n,
				))
			}
		}

		// Remove any ".total" suffix, this is redundant for Prometheus.
		if strings.HasSuffix(n, ".total") {
			n = n[:len(n)-len(".total")]
		}

		opts := []instrument.Option{
			instrument.WithUnit(unit.Unit(u)),
			instrument.WithDescription(m.Description),
		}
		var inst instrument.Asynchronous
		var err error
		if m.Cumulative {
			switch m.Kind {
			case metrics.KindUint64:
				inst, err = r.meter.AsyncInt64().Counter(n, opts...)
			case metrics.KindFloat64:
				inst, err = r.meter.AsyncFloat64().Counter(n, opts...)
			case metrics.KindFloat64Histogram:
				// Not implemented Histogram[float64]
				continue
			}
		} else {
			switch m.Kind {
			case metrics.KindUint64:
				inst, err = r.meter.AsyncInt64().UpDownCounter(n, opts...)
			case metrics.KindFloat64:
				// Note: this has never been used.
				inst, err = r.meter.AsyncFloat64().Gauge(n, opts...)
			case metrics.KindFloat64Histogram:
				// Not implemented GaugeHistogram[float64]
				continue
			}
		}
		if err != nil {
			rerr = multierror.Append(rerr, err)
		}

		samp := metrics.Sample{
			Name: m.Name,
		}
		r.samples = append(r.samples, samp)
		r.instruments = append(r.instruments, inst)
	}

	if err := r.meter.RegisterCallback(r.instruments, func(ctx context.Context) {
		metrics.Read(r.samples)
		for idx, samp := range r.samples {
			switch samp.Value.Kind() {
			case metrics.KindUint64:
				r.instruments[idx].(int64Observer).Observe(ctx, int64(samp.Value.Uint64()))
			case metrics.KindFloat64:
				r.instruments[idx].(float64Observer).Observe(ctx, samp.Value.Float64())
			default:
				// KindFloat64Histogram (unsupported in OTel) and KindBad
				// (unsupported by runtime/metrics).  Neither should happen
				// if runtime/metrics and the code above are working correctly.
				otel.Handle(fmt.Errorf("invalid runtime/metrics value kind: %v", samp.Value.Kind()))
			}
		}
	}); err != nil {
		rerr = multierror.Append(rerr, err)
	}
	return rerr
}
