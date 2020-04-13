package runtime

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
	"golang.org/x/time/rate"

	"github.com/open-telemetry/opentelemetry-go-contrib/plugins/metrics/runtime/internal/metrics"
)

var (
	initialisation time.Time
)

func init() {
	initialisation = time.Now()
}

// various runtime metrics have a certain overhead, and as such, we only measure them on a defined interval
func Runtime(ctx context.Context, meter metric.Meter, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	ctxTimeout, _ := context.WithTimeout(ctx, time.Second)
	var m, p metrics.Metrics
	m, _ = metrics.Measure(ctxTimeout)

	if interval == 0 {
		interval = time.Second
	}

	go func() {
		limit := rate.NewLimiter(rate.Every(time.Second), 1)
		for {
			err := limit.Wait(ctx)

			if err != nil {
				break
			}

			ctxTimeout, _ := context.WithTimeout(ctx, time.Second)
			// store the previous measurement for reporting deltas
			p = m
			// we assume here we are replacing m and thus do ont require a concurrency lock
			m, _ = metrics.Measure(ctxTimeout)

			metric.Must(meter).NewFloat64Counter("runtime.go.cpu.user").Bind().Add(ctx,
				m.ProcessCPU.User-p.ProcessCPU.User,
			)

			metric.Must(meter).NewFloat64Counter("runtime.go.cpu.sys").Bind().Add(ctx,
				m.ProcessCPU.System-p.ProcessCPU.System,
			)

			metric.Must(meter).NewInt64Counter("runtime.go.gc.count",
				metric.WithDescription("counter of completed GC cycles")).Bind().Add(ctx,
				int64(m.Runtime.NumGC-p.Runtime.NumGC),
			)
		}
	}()

	metric.Must(meter).RegisterInt64Observer("runtime.uptime",
		func(result metric.Int64ObserverResult) {
			result.Observe(time.Since(initialisation).Milliseconds())
		},
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("milliseconds since application was initialized"),
	)

	metric.Must(meter).RegisterInt64Observer("runtime.go.goroutine", func(result metric.Int64ObserverResult) {
		result.Observe(int64(m.Runtime.NumGoroutine))
	})

	metric.Must(meter).RegisterInt64Observer("mem.available", func(result metric.Int64ObserverResult) {
		result.Observe(int64(m.Memory.Available))
	})

	metric.Must(meter).RegisterInt64Observer("mem.total", func(result metric.Int64ObserverResult) {
		result.Observe(int64(m.Memory.Total))
	})

	metric.Must(meter).RegisterInt64Observer("runtime.go.mem.heap_alloc", func(result metric.Int64ObserverResult) {
		result.Observe(int64(m.Memory.HeapAlloc))
	}, metric.WithUnit(unit.Bytes), metric.WithDescription("bytes of allocated heap objects"))

	return cancel
}
