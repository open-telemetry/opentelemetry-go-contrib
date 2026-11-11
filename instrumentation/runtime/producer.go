// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"errors"
	"math"
	"runtime/metrics"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
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
// The following opt-in metrics are produced when listed, comma-separated, in
// the OTEL_GO_X_RUNTIME_METRICS_OPTIN environment variable:
//
//	go.memory.gc.pause.duration  s        Distribution of individual GC-related stop-the-world pause latencies.
func NewProducer(opts ...ProducerOption) *Producer {
	c := newProducerConfig(opts...)
	histogramMetrics := []string{goSchedLatencies}
	enableGCPauses := optInGCPauses.enabled()
	if enableGCPauses {
		histogramMetrics = append(histogramMetrics, goGCPauses)
	}
	return &Producer{
		collector:      newCollector(c.MinimumRefreshInterval, histogramMetrics),
		enableGCPauses: enableGCPauses,
	}
}

// Produce returns precomputed histogram metrics from the go runtime, or an error if unsuccessful.
func (p *Producer) Produce(context.Context) ([]metricdata.ScopeMetrics, error) {
	p.lock.Lock()
	p.collector.refresh()
	// The runtime reuses histogram memory on later reads, so copy it while
	// holding the lock to keep returned datapoints stable.
	schedHist := cloneHistogram(p.collector.getHistogram(goSchedLatencies))
	pauseHist := cloneHistogram(p.collector.getHistogram(goGCPauses))
	// Use the last collection time (which may or may not be now) for the timestamp.
	collectedAt := p.collector.lastCollect
	p.lock.Unlock()
	schedDp := convertRuntimeHistogram(schedHist, collectedAt)
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
	// go.memory.gc.pause.duration is opt-in, so it is skipped instead of
	// failing the collection of the always-on go.schedule.duration metric.
	if pauseDp := convertRuntimeHistogram(pauseHist, collectedAt); p.enableGCPauses && len(pauseDp) > 0 {
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
				Name:      ScopeName,
				Version:   Version,
				SchemaURL: semconv.SchemaURL,
			},
			Metrics: scopeMetrics,
		},
	}, nil
}

var emptySet = attribute.EmptySet()

func cloneHistogram(h *metrics.Float64Histogram) *metrics.Float64Histogram {
	if h == nil {
		return nil
	}
	return &metrics.Float64Histogram{
		Counts:  slices.Clone(h.Counts),
		Buckets: slices.Clone(h.Buckets),
	}
}

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
