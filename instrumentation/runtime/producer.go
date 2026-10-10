// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"errors"
	"math"
	"runtime/metrics"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/semconv/v1.43.0/goconv"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

// Producer is a metric.Producer, which provides precomputed histogram metrics from the go runtime.
type Producer struct {
	lock           sync.Mutex
	collector      *goCollector
	enableGCPauses bool
}

var _ metric.Producer = (*Producer)(nil)

// NewProducer creates a Producer which provides precomputed histogram metrics from the go runtime.
//
// Metrics emitted by NewProducer include:
//
//	go.schedule.duration    s             The time goroutines have spent in the scheduler in a runnable state before actually running.
//
// The following opt-in metrics are produced when enabled with
// [WithOptInMetrics] or listed, comma-separated, in the
// OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable:
//
//	go.memory.gc.pause.duration  s        Distribution of individual GC-related stop-the-world pause latencies.
func NewProducer(opts ...ProducerOption) *Producer {
	c := newProducerConfig(opts...)
	histogramMetrics := []string{goSchedLatencies}
	enableGCPauses := c.optInEnabled(MemoryGCPauseDuration)
	if enableGCPauses {
		histogramMetrics = append(histogramMetrics, goGCPauses)
	}
	return &Producer{
		collector:      newCollector(c.MinimumReadMemStatsInterval, histogramMetrics),
		enableGCPauses: enableGCPauses,
	}
}

// Produce returns precomputed histogram metrics from the go runtime, or an error if unsuccessful.
func (p *Producer) Produce(context.Context) ([]metricdata.ScopeMetrics, error) {
	p.lock.Lock()
	p.collector.refresh()
	schedHist := p.collector.getHistogram(goSchedLatencies)
	pauseHist := p.collector.getHistogram(goGCPauses)
	p.lock.Unlock()
	// Use the last collection time (which may or may not be now) for the timestamp.
	schedDp := convertRuntimeHistogram(schedHist, p.collector.lastCollect)
	if len(schedDp) == 0 {
		return nil, errors.New("unable to obtain go.schedule.duration metric from the runtime")
	}
	scopeMetrics := []metricdata.Metrics{
		{
			Name:        goconv.ScheduleDuration{}.Name(),
			Description: goconv.ScheduleDuration{}.Description(),
			Unit:        goconv.ScheduleDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints:  schedDp,
			},
		},
	}
	if p.enableGCPauses {
		pauseDp := convertRuntimeHistogram(pauseHist, p.collector.lastCollect)
		if len(pauseDp) == 0 {
			return nil, errors.New("unable to obtain go.memory.gc.pause.duration metric from the runtime")
		}
		scopeMetrics = append(scopeMetrics, metricdata.Metrics{
			Name:        goconv.MemoryGCPauseDuration{}.Name(),
			Description: goconv.MemoryGCPauseDuration{}.Description(),
			Unit:        goconv.MemoryGCPauseDuration{}.Unit(),
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints:  pauseDp,
			},
		})
	}
	return []metricdata.ScopeMetrics{
		{
			Scope: instrumentation.Scope{
				Name:    ScopeName,
				Version: Version,
			},
			Metrics: scopeMetrics,
		},
	}, nil
}

var emptySet = attribute.EmptySet()

func convertRuntimeHistogram(runtimeHist *metrics.Float64Histogram, ts time.Time) []metricdata.HistogramDataPoint[float64] {
	if runtimeHist == nil {
		return nil
	}
	bounds := runtimeHist.Buckets
	counts := runtimeHist.Counts
	if len(bounds) < 2 {
		// runtime histograms are guaranteed to have at least two bucket boundaries.
		return nil
	}
	// trim the first bucket since it is a lower bound. OTel histogram boundaries only have an upper bound.
	bounds = bounds[1:]
	if bounds[len(bounds)-1] == math.Inf(1) {
		// trim the last bucket if it is +Inf, since the +Inf boundary is implicit in OTel.
		bounds = bounds[:len(bounds)-1]
	} else {
		// if the last bucket is not +Inf, append an extra zero count since
		// the implicit +Inf bucket won't have any observations.
		counts = append(counts, 0)
	}
	count := uint64(0)
	sum := float64(0)
	for i, c := range counts {
		count += c
		// This computed sum is an underestimate, since it assumes each
		// observation happens at the bucket's lower bound.
		if i > 0 && c != 0 {
			sum += bounds[i-1] * float64(c)
		}
	}

	return []metricdata.HistogramDataPoint[float64]{
		{
			StartTime:    startTime,
			Count:        count,
			Sum:          sum,
			Time:         ts,
			Bounds:       bounds,
			BucketCounts: counts,
			Attributes:   *emptySet,
		},
	}
}
