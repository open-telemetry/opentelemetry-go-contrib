package runtime

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/metric"
	"golang.org/x/time/rate"

	"github.com/open-telemetry/opentelemetry-go-contrib/plugins/metrics/runtime/internal/metrics"
)

func Runtime(ctx context.Context, meter metric.Meter) func() {
	ctx, cancel := context.WithCancel(ctx)

	lock := &sync.Mutex{}
	m, _  := metrics.Measure(ctx)

	go func() {
		limit := rate.NewLimiter(rate.Every(time.Second), 1)
		for {
			err := limit.Wait(ctx)

			if err != nil {
				break
			}

			lock.Lock()
			m, _ = metrics.Measure(ctx)
			lock.Unlock()
		}
	}()

	callBack := func(result metric.Int64ObserverResult) {
		lock.Lock()
		result.Observe(int64(m.Runtime.NumGoroutine))
		lock.Unlock()
	}

	metric.Must(meter).RegisterInt64Observer("runtime.go.goroutine", callBack)

	return func() {
		cancel()
	}
}
