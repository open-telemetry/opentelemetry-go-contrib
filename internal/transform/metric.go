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

// Package transform provides translations for opentelemetry-go concepts and
// structures to otlp structures.
package transform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	commonpb "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	metricpb "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	resourcepb "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"

	"go.opentelemetry.io/otel/api/label"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	// ErrUnimplementedAgg is returned when a transformation of an unimplemented
	// aggregator is attempted.
	ErrUnimplementedAgg = errors.New("unimplemented aggregator")

	// ErrUnknownValueType is returned when a transformation of an unknown value
	// is attempted.
	ErrUnknownValueType = errors.New("invalid value type")

	// ErrContextCanceled is returned when a context cancellation halts a
	// transformation.
	ErrContextCanceled = errors.New("context canceled")

	// ErrTransforming is returned when an unexected error is encoutered transforming.
	ErrTransforming = errors.New("transforming failed")
)

// result is the product of transforming Records into OTLP Metrics.
type result struct {
	Resource               *resource.Resource
	InstrumentationLibrary instrumentation.Library
	Metric                 *metricpb.Metric
	Err                    error
}

// CheckpointSet transforms all records contained in a checkpoint into
// batched OTLP ResourceMetrics.
func CheckpointSet(ctx context.Context, exportSelector export.ExportKindSelector, cps export.CheckpointSet, numWorkers uint) ([]*metricpb.ResourceMetrics, error) {
	records, errc := source(ctx, exportSelector, cps)

	// Start a fixed number of goroutines to transform records.
	transformed := make(chan result)
	var wg sync.WaitGroup
	wg.Add(int(numWorkers))
	for i := uint(0); i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			transformer(ctx, records, transformed)
		}()
	}
	go func() {
		wg.Wait()
		close(transformed)
	}()

	// Synchronously collect the transformed records and transmit.
	rms, err := sink(ctx, transformed)
	if err != nil {
		return nil, err
	}

	// source is complete, check for any errors.
	if err := <-errc; err != nil {
		return nil, err
	}
	return rms, nil
}

// source starts a goroutine that sends each one of the Records yielded by
// the CheckpointSet on the returned chan. Any error encoutered will be sent
// on the returned error chan after seeding is complete.
func source(ctx context.Context, exportSelector export.ExportKindSelector, cps export.CheckpointSet) (<-chan export.Record, <-chan error) {
	errc := make(chan error, 1)
	out := make(chan export.Record)
	// Seed records into process.
	go func() {
		defer close(out)
		// No select is needed since errc is buffered.
		errc <- cps.ForEach(exportSelector, func(r export.Record) error {
			select {
			case <-ctx.Done():
				return ErrContextCanceled
			case out <- r:
			}
			return nil
		})
	}()
	return out, errc
}

// transformer transforms records read from the passed in chan into
// OTLP Metrics which are sent on the out chan.
func transformer(ctx context.Context, in <-chan export.Record, out chan<- result) {
	for r := range in {
		m, err := Record(r)
		// Propagate errors, but do not send empty results.
		if err == nil && m == nil {
			continue
		}
		res := result{
			Resource: r.Resource(),
			InstrumentationLibrary: instrumentation.Library{
				Name:    r.Descriptor().InstrumentationName(),
				Version: r.Descriptor().InstrumentationVersion(),
			},
			Metric: m,
			Err:    err,
		}
		select {
		case <-ctx.Done():
			return
		case out <- res:
		}
	}
}

// sink collects transformed Records and batches them.
//
// Any errors encoutered transforming input will be reported with an
// ErrTransforming as well as the completed ResourceMetrics. It is up to the
// caller to handle any incorrect data in these ResourceMetrics.
func sink(ctx context.Context, in <-chan result) ([]*metricpb.ResourceMetrics, error) {
	var errStrings []string

	type resourceBatch struct {
		Resource *resourcepb.Resource
		// Group by instrumentation library name and then the MetricDescriptor.
		InstrumentationLibraryBatches map[instrumentation.Library]map[string]*metricpb.Metric
	}

	// group by unique Resource string.
	grouped := make(map[label.Distinct]resourceBatch)
	for res := range in {
		if res.Err != nil {
			errStrings = append(errStrings, res.Err.Error())
			continue
		}

		rID := res.Resource.Equivalent()
		rb, ok := grouped[rID]
		if !ok {
			rb = resourceBatch{
				Resource:                      Resource(res.Resource),
				InstrumentationLibraryBatches: make(map[instrumentation.Library]map[string]*metricpb.Metric),
			}
			grouped[rID] = rb
		}

		mb, ok := rb.InstrumentationLibraryBatches[res.InstrumentationLibrary]
		if !ok {
			mb = make(map[string]*metricpb.Metric)
			rb.InstrumentationLibraryBatches[res.InstrumentationLibrary] = mb
		}

		mID := res.Metric.GetMetricDescriptor().String()
		m, ok := mb[mID]
		if !ok {
			mb[mID] = res.Metric
			continue
		}
		if len(res.Metric.Int64DataPoints) > 0 {
			m.Int64DataPoints = append(m.Int64DataPoints, res.Metric.Int64DataPoints...)
		}
		if len(res.Metric.DoubleDataPoints) > 0 {
			m.DoubleDataPoints = append(m.DoubleDataPoints, res.Metric.DoubleDataPoints...)
		}
		if len(res.Metric.HistogramDataPoints) > 0 {
			m.HistogramDataPoints = append(m.HistogramDataPoints, res.Metric.HistogramDataPoints...)
		}
		if len(res.Metric.SummaryDataPoints) > 0 {
			m.SummaryDataPoints = append(m.SummaryDataPoints, res.Metric.SummaryDataPoints...)
		}
	}

	if len(grouped) == 0 {
		return nil, nil
	}

	var rms []*metricpb.ResourceMetrics
	for _, rb := range grouped {
		rm := &metricpb.ResourceMetrics{Resource: rb.Resource}
		for il, mb := range rb.InstrumentationLibraryBatches {
			ilm := &metricpb.InstrumentationLibraryMetrics{
				Metrics: make([]*metricpb.Metric, 0, len(mb)),
			}
			if il != (instrumentation.Library{}) {
				ilm.InstrumentationLibrary = &commonpb.InstrumentationLibrary{
					Name:    il.Name,
					Version: il.Version,
				}
			}
			for _, m := range mb {
				ilm.Metrics = append(ilm.Metrics, m)
			}
			rm.InstrumentationLibraryMetrics = append(rm.InstrumentationLibraryMetrics, ilm)
		}
		rms = append(rms, rm)
	}

	// Report any transform errors.
	if len(errStrings) > 0 {
		return rms, fmt.Errorf("%w:\n -%s", ErrTransforming, strings.Join(errStrings, "\n -"))
	}
	return rms, nil
}

// Record transforms a Record into an OTLP Metric. An ErrUnimplementedAgg
// error is returned if the Record Aggregator is not supported.
func Record(r export.Record) (*metricpb.Metric, error) {
	switch a := r.Aggregation().(type) {
	case aggregation.MinMaxSumCount:
		return minMaxSumCount(r, a)
	case aggregation.Sum:
		return sum(r, a)
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnimplementedAgg, a)
	}
}

// sum transforms a Sum Aggregator into an OTLP Metric.
func sum(record export.Record, a aggregation.Sum) (*metricpb.Metric, error) {
	desc := record.Descriptor()
	labels := record.Labels()
	sum, err := a.Sum()
	if err != nil {
		return nil, err
	}

	m := &metricpb.Metric{
		MetricDescriptor: &metricpb.MetricDescriptor{
			Name:        desc.Name(),
			Description: desc.Description(),
			Unit:        string(desc.Unit()),
		},
	}

	switch n := desc.NumberKind(); n {
	case metric.Int64NumberKind:
		m.MetricDescriptor.Type = metricpb.MetricDescriptor_INT64
		m.Int64DataPoints = []*metricpb.Int64DataPoint{
			{
				Value:             sum.CoerceToInt64(n),
				Labels:            stringKeyValues(labels.Iter()),
				StartTimeUnixNano: uint64(record.StartTime().UnixNano()),
				TimeUnixNano:      uint64(record.EndTime().UnixNano()),
			},
		}
	case metric.Float64NumberKind:
		m.MetricDescriptor.Type = metricpb.MetricDescriptor_DOUBLE
		m.DoubleDataPoints = []*metricpb.DoubleDataPoint{
			{
				Value:             sum.CoerceToFloat64(n),
				Labels:            stringKeyValues(labels.Iter()),
				StartTimeUnixNano: uint64(record.StartTime().UnixNano()),
				TimeUnixNano:      uint64(record.EndTime().UnixNano()),
			},
		}
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnknownValueType, n)
	}

	return m, nil
}

// minMaxSumCountValue returns the values of the MinMaxSumCount Aggregator
// as discret values.
func minMaxSumCountValues(a aggregation.MinMaxSumCount) (min, max, sum metric.Number, count int64, err error) {
	if min, err = a.Min(); err != nil {
		return
	}
	if max, err = a.Max(); err != nil {
		return
	}
	if sum, err = a.Sum(); err != nil {
		return
	}
	if count, err = a.Count(); err != nil {
		return
	}
	return
}

// minMaxSumCount transforms a MinMaxSumCount Aggregator into an OTLP Metric.
func minMaxSumCount(record export.Record, a aggregation.MinMaxSumCount) (*metricpb.Metric, error) {
	desc := record.Descriptor()
	labels := record.Labels()
	min, max, sum, count, err := minMaxSumCountValues(a)
	if err != nil {
		return nil, err
	}

	numKind := desc.NumberKind()
	return &metricpb.Metric{
		MetricDescriptor: &metricpb.MetricDescriptor{
			Name:        desc.Name(),
			Description: desc.Description(),
			Unit:        string(desc.Unit()),
			Type:        metricpb.MetricDescriptor_SUMMARY,
		},
		SummaryDataPoints: []*metricpb.SummaryDataPoint{
			{
				Labels: stringKeyValues(labels.Iter()),
				Count:  uint64(count),
				Sum:    sum.CoerceToFloat64(numKind),
				PercentileValues: []*metricpb.SummaryDataPoint_ValueAtPercentile{
					{
						Percentile: 0.0,
						Value:      min.CoerceToFloat64(numKind),
					},
					{
						Percentile: 100.0,
						Value:      max.CoerceToFloat64(numKind),
					},
				},
				StartTimeUnixNano: uint64(record.StartTime().UnixNano()),
				TimeUnixNano:      uint64(record.EndTime().UnixNano()),
			},
		},
	}, nil
}

// stringKeyValues transforms a label iterator into an OTLP StringKeyValues.
func stringKeyValues(iter label.Iterator) []*commonpb.StringKeyValue {
	l := iter.Len()
	if l == 0 {
		return nil
	}
	result := make([]*commonpb.StringKeyValue, 0, l)
	for iter.Next() {
		kv := iter.Label()
		result = append(result, &commonpb.StringKeyValue{
			Key:   string(kv.Key),
			Value: kv.Value.Emit(),
		})
	}
	return result
}
