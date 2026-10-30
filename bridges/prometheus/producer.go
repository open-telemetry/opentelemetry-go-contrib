// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace"
)

const (
	scopeName           = "go.opentelemetry.io/contrib/bridges/prometheus"
	traceIDLabel        = "trace_id"
	spanIDLabel         = "span_id"
	scopeLabelPrefix    = "otel_scope_"
	scopeNameLabel      = scopeLabelPrefix + "name"
	scopeVersionLabel   = scopeLabelPrefix + "version"
	scopeSchemaURLLabel = scopeLabelPrefix + "schema_url"
)

var (
	errUnsupportedType = errors.New("unsupported metric type")
	processStartTime   = time.Now()
	defaultScope       = instrumentation.Scope{
		Name:    scopeName,
		Version: Version,
	}
)

// Producer is a [metric.Producer] that fetches metrics from Prometheus.
type Producer struct {
	gatherers prometheus.Gatherers
}

var _ metric.Producer = (*Producer)(nil)

// NewMetricProducer returns a [Producer] that fetches metrics from
// Prometheus. This can be used to allow Prometheus instrumentation to be
// added to an OpenTelemetry export pipeline.
func NewMetricProducer(opts ...Option) *Producer {
	cfg := newConfig(opts...)
	return &Producer{
		gatherers: cfg.gatherers,
	}
}

// Produce fetches metrics from the configured Prometheus gatherers and
// converts them to OpenTelemetry metrics.
func (p *Producer) Produce(context.Context) ([]metricdata.ScopeMetrics, error) {
	now := time.Now()
	var errs multierr
	var c scopeMetricsCollector
	for _, gatherer := range p.gatherers {
		promMetrics, err := gatherer.Gather()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := c.convertPrometheusMetrics(promMetrics, now); err != nil {
			errs = append(errs, err)
		}
	}
	if errs.errOrNil() != nil {
		otel.Handle(errs.errOrNil())
	}
	return c.scopeMetrics, nil
}

type scopeMetricsCollector struct {
	scopeMetrics []metricdata.ScopeMetrics
	scopeIndices map[instrumentation.Scope]int
}

func (c *scopeMetricsCollector) appendMetric(scope instrumentation.Scope, m metricdata.Metrics) {
	if c.scopeIndices == nil {
		if len(c.scopeMetrics) == 0 {
			c.scopeMetrics = []metricdata.ScopeMetrics{{
				Scope:   scope,
				Metrics: []metricdata.Metrics{m},
			}}
			return
		}
		if c.scopeMetrics[0].Scope == scope {
			c.scopeMetrics[0].Metrics = append(c.scopeMetrics[0].Metrics, m)
			return
		}
		c.scopeIndices = map[instrumentation.Scope]int{
			c.scopeMetrics[0].Scope: 0,
		}
	}
	idx, ok := c.scopeIndices[scope]
	if !ok {
		idx = len(c.scopeMetrics)
		c.scopeIndices[scope] = idx
		c.scopeMetrics = append(c.scopeMetrics, metricdata.ScopeMetrics{
			Scope: scope,
		})
	}
	c.scopeMetrics[idx].Metrics = append(c.scopeMetrics[idx].Metrics, m)
}

func (c *scopeMetricsCollector) convertPrometheusMetrics(promMetrics []*dto.MetricFamily, now time.Time) error {
	var errs multierr
	for _, pm := range promMetrics {
		if len(pm.GetMetric()) == 0 {
			// This shouldn't ever happen
			continue
		}
		switch pm.GetType() {
		case dto.MetricType_GAUGE, dto.MetricType_UNTYPED, dto.MetricType_COUNTER, dto.MetricType_SUMMARY, dto.MetricType_HISTOGRAM:
		default:
			// MetricType_GAUGE_HISTOGRAM
			errs = append(errs, fmt.Errorf("%w: %v for metric %v", errUnsupportedType, pm.GetType(), pm.GetName()))
			continue
		}
		unit := unitWordToUCUM(pm.GetUnit())
		for _, group := range groupByScope(pm.GetMetric()) {
			newMetric := metricdata.Metrics{
				Name:        pm.GetName(),
				Description: pm.GetHelp(),
				Unit:        unit,
			}
			switch pm.GetType() {
			case dto.MetricType_GAUGE:
				newMetric.Data = convertGauge(group.metrics, now)
			case dto.MetricType_UNTYPED:
				newMetric.Data = convertUntyped(group.metrics, now)
			case dto.MetricType_COUNTER:
				newMetric.Data = convertCounter(group.metrics, now)
			case dto.MetricType_SUMMARY:
				newMetric.Data = convertSummary(group.metrics, now)
			case dto.MetricType_HISTOGRAM:
				if isExponentialHistogram(group.metrics[0].GetHistogram()) {
					newMetric.Data = convertExponentialHistogram(group.metrics, now)
				} else {
					newMetric.Data = convertHistogram(group.metrics, now)
				}
			}
			c.appendMetric(group.scope, newMetric)
		}
	}
	return errs.errOrNil()
}

func isExponentialHistogram(hist *dto.Histogram) bool {
	// The prometheus go client ensures at least one of these is non-zero
	// so it can be distinguished from a fixed-bucket histogram.
	// https://github.com/prometheus/client_golang/blob/7ac90362b02729a65109b33d172bafb65d7dab50/prometheus/histogram.go#L818
	return hist.GetZeroThreshold() > 0 ||
		hist.GetZeroCount() > 0 ||
		len(hist.GetPositiveSpan()) > 0 ||
		len(hist.GetNegativeSpan()) > 0
}

func convertGauge(metrics []*dto.Metric, now time.Time) metricdata.Gauge[float64] {
	otelGauge := metricdata.Gauge[float64]{
		DataPoints: make([]metricdata.DataPoint[float64], len(metrics)),
	}
	for i, m := range metrics {
		dp := metricdata.DataPoint[float64]{
			Attributes: convertLabels(m.GetLabel()),
			Time:       now,
			Value:      m.GetGauge().GetValue(),
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelGauge.DataPoints[i] = dp
	}
	return otelGauge
}

func convertUntyped(metrics []*dto.Metric, now time.Time) metricdata.Gauge[float64] {
	otelGauge := metricdata.Gauge[float64]{
		DataPoints: make([]metricdata.DataPoint[float64], len(metrics)),
	}
	for i, m := range metrics {
		dp := metricdata.DataPoint[float64]{
			Attributes: convertLabels(m.GetLabel()),
			Time:       now,
			Value:      m.GetUntyped().GetValue(),
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelGauge.DataPoints[i] = dp
	}
	return otelGauge
}

func convertCounter(metrics []*dto.Metric, now time.Time) metricdata.Sum[float64] {
	otelCounter := metricdata.Sum[float64]{
		DataPoints:  make([]metricdata.DataPoint[float64], len(metrics)),
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: true,
	}
	for i, m := range metrics {
		dp := metricdata.DataPoint[float64]{
			Attributes: convertLabels(m.GetLabel()),
			StartTime:  processStartTime,
			Time:       now,
			Value:      m.GetCounter().GetValue(),
		}
		if ex := m.GetCounter().GetExemplar(); ex != nil {
			dp.Exemplars = []metricdata.Exemplar[float64]{convertExemplar(ex)}
		}
		createdTs := m.GetCounter().GetCreatedTimestamp()
		if createdTs.IsValid() {
			dp.StartTime = createdTs.AsTime()
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelCounter.DataPoints[i] = dp
	}
	return otelCounter
}

func convertExponentialHistogram(metrics []*dto.Metric, now time.Time) metricdata.ExponentialHistogram[float64] {
	otelExpHistogram := metricdata.ExponentialHistogram[float64]{
		DataPoints:  make([]metricdata.ExponentialHistogramDataPoint[float64], len(metrics)),
		Temporality: metricdata.CumulativeTemporality,
	}
	for i, m := range metrics {
		dp := metricdata.ExponentialHistogramDataPoint[float64]{
			Attributes:    convertLabels(m.GetLabel()),
			StartTime:     processStartTime,
			Time:          now,
			Count:         m.GetHistogram().GetSampleCount(),
			Sum:           m.GetHistogram().GetSampleSum(),
			Scale:         m.GetHistogram().GetSchema(),
			ZeroCount:     m.GetHistogram().GetZeroCount(),
			ZeroThreshold: m.GetHistogram().GetZeroThreshold(),
			PositiveBucket: convertExponentialBuckets(
				m.GetHistogram().GetPositiveSpan(),
				m.GetHistogram().GetPositiveDelta(),
			),
			NegativeBucket: convertExponentialBuckets(
				m.GetHistogram().GetNegativeSpan(),
				m.GetHistogram().GetNegativeDelta(),
			),
		}
		if exs := m.GetHistogram().GetExemplars(); len(exs) > 0 {
			dp.Exemplars = make([]metricdata.Exemplar[float64], len(exs))
			for j, ex := range exs {
				dp.Exemplars[j] = convertExemplar(ex)
			}
		}
		createdTs := m.GetHistogram().GetCreatedTimestamp()
		if createdTs.IsValid() {
			dp.StartTime = createdTs.AsTime()
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelExpHistogram.DataPoints[i] = dp
	}
	return otelExpHistogram
}

func convertExponentialBuckets(bucketSpans []*dto.BucketSpan, deltas []int64) metricdata.ExponentialBucket {
	if len(bucketSpans) == 0 {
		return metricdata.ExponentialBucket{}
	}
	// Prometheus Native Histograms buckets are indexed by upper boundary
	// while Exponential Histograms are indexed by lower boundary, the result
	// being that the Offset fields are different-by-one.
	initialOffset := bucketSpans[0].GetOffset() - 1
	// We will have one bucket count for each delta, and zeros for the offsets
	// after the initial offset.
	lenCounts := len(deltas)
	for i, bs := range bucketSpans {
		if i != 0 {
			lenCounts += int(bs.GetOffset())
		}
	}
	counts := make([]uint64, lenCounts)
	deltaIndex := 0
	countIndex := int32(0)
	count := int64(0)
	for i, bs := range bucketSpans {
		// Do not insert zeroes if this is the first bucketSpan, since those
		// zeroes are accounted for in the Offset field.
		if i != 0 {
			// Increase the count index by the Offset to insert Offset zeroes
			countIndex += bs.GetOffset()
		}
		for range bs.GetLength() {
			// Convert deltas to the cumulative number of observations
			count += deltas[deltaIndex]
			deltaIndex++
			// count should always be positive after accounting for deltas
			if count > 0 {
				counts[countIndex] = uint64(count)
			}
			countIndex++
		}
	}
	return metricdata.ExponentialBucket{
		Offset: initialOffset,
		Counts: counts,
	}
}

func convertHistogram(metrics []*dto.Metric, now time.Time) metricdata.Histogram[float64] {
	otelHistogram := metricdata.Histogram[float64]{
		DataPoints:  make([]metricdata.HistogramDataPoint[float64], len(metrics)),
		Temporality: metricdata.CumulativeTemporality,
	}
	for i, m := range metrics {
		bounds, bucketCounts, exemplars := convertBuckets(m.GetHistogram().GetBucket(), m.GetHistogram().GetSampleCount())
		dp := metricdata.HistogramDataPoint[float64]{
			Attributes:   convertLabels(m.GetLabel()),
			StartTime:    processStartTime,
			Time:         now,
			Count:        m.GetHistogram().GetSampleCount(),
			Sum:          m.GetHistogram().GetSampleSum(),
			Bounds:       bounds,
			BucketCounts: bucketCounts,
			Exemplars:    exemplars,
		}
		createdTs := m.GetHistogram().GetCreatedTimestamp()
		if createdTs.IsValid() {
			dp.StartTime = createdTs.AsTime()
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelHistogram.DataPoints[i] = dp
	}
	return otelHistogram
}

func convertBuckets(buckets []*dto.Bucket, sampleCount uint64) ([]float64, []uint64, []metricdata.Exemplar[float64]) {
	if len(buckets) == 0 {
		// This should never happen
		return nil, nil, nil
	}
	// buckets will only include the +Inf bucket if there is an exemplar for it
	// https://github.com/prometheus/client_golang/blob/d038ab96c0c7b9cd217a39072febd610bcdf1fd8/prometheus/metric.go#L189
	// we need to handle the case where it is present, or where it is missing.
	hasInf := math.IsInf(buckets[len(buckets)-1].GetUpperBound(), +1)
	var bounds []float64
	var bucketCounts []uint64
	if hasInf {
		bounds = make([]float64, len(buckets)-1)
		bucketCounts = make([]uint64, len(buckets))
	} else {
		bounds = make([]float64, len(buckets))
		bucketCounts = make([]uint64, len(buckets)+1)
	}
	exemplars := make([]metricdata.Exemplar[float64], 0)
	var previousCount uint64
	for i, bucket := range buckets {
		// The last bound may be the +Inf bucket, which is implied in OTel, but
		// is explicit in Prometheus. Skip the last boundary if it is the +Inf
		// bound.
		if bound := bucket.GetUpperBound(); !math.IsInf(bound, +1) {
			bounds[i] = bound
		}
		previousCount, bucketCounts[i] = bucket.GetCumulativeCount(), bucket.GetCumulativeCount()-previousCount
		if ex := bucket.GetExemplar(); ex != nil {
			exemplars = append(exemplars, convertExemplar(ex))
		}
	}
	if !hasInf {
		// The Inf bucket was missing, so set the last bucket counts to the
		// overall count
		bucketCounts[len(bucketCounts)-1] = sampleCount - previousCount
	}
	return bounds, bucketCounts, exemplars
}

func convertSummary(metrics []*dto.Metric, now time.Time) metricdata.Summary {
	otelSummary := metricdata.Summary{
		DataPoints: make([]metricdata.SummaryDataPoint, len(metrics)),
	}
	for i, m := range metrics {
		dp := metricdata.SummaryDataPoint{
			Attributes:     convertLabels(m.GetLabel()),
			StartTime:      processStartTime,
			Time:           now,
			Count:          m.GetSummary().GetSampleCount(),
			Sum:            m.GetSummary().GetSampleSum(),
			QuantileValues: convertQuantiles(m.GetSummary().GetQuantile()),
		}
		createdTs := m.GetSummary().GetCreatedTimestamp()
		if createdTs.IsValid() {
			dp.StartTime = createdTs.AsTime()
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelSummary.DataPoints[i] = dp
	}
	return otelSummary
}

func convertQuantiles(quantiles []*dto.Quantile) []metricdata.QuantileValue {
	otelQuantiles := make([]metricdata.QuantileValue, len(quantiles))
	for i, quantile := range quantiles {
		dp := metricdata.QuantileValue{
			Quantile: quantile.GetQuantile(),
			Value:    quantile.GetValue(),
		}
		otelQuantiles[i] = dp
	}
	return otelQuantiles
}

type scopedMetrics struct {
	scope   instrumentation.Scope
	metrics []*dto.Metric
}

func groupByScope(metrics []*dto.Metric) []scopedMetrics {
	firstScope := extractScope(metrics[0].GetLabel())
	var groups []scopedMetrics
	var scopeIndices map[instrumentation.Scope]int
	for i := 1; i < len(metrics); i++ {
		s := extractScope(metrics[i].GetLabel())
		if groups == nil {
			if s == firstScope {
				continue
			}
			groups = []scopedMetrics{
				{scope: firstScope, metrics: append([]*dto.Metric(nil), metrics[:i]...)},
				{scope: s, metrics: []*dto.Metric{metrics[i]}},
			}
			scopeIndices = map[instrumentation.Scope]int{
				firstScope: 0,
				s:          1,
			}
			continue
		}
		if idx, ok := scopeIndices[s]; ok {
			groups[idx].metrics = append(groups[idx].metrics, metrics[i])
		} else {
			scopeIndices[s] = len(groups)
			groups = append(groups, scopedMetrics{
				scope:   s,
				metrics: []*dto.Metric{metrics[i]},
			})
		}
	}
	if groups == nil {
		return []scopedMetrics{{scope: firstScope, metrics: metrics}}
	}
	return groups
}

func extractScope(labels []*dto.LabelPair) instrumentation.Scope {
	var scope instrumentation.Scope
	var scopeAttrs []attribute.KeyValue
	hasScope := false
	for _, l := range labels {
		name := l.GetName()
		if !strings.HasPrefix(name, scopeLabelPrefix) {
			continue
		}
		hasScope = true
		switch name {
		case scopeNameLabel:
			scope.Name = l.GetValue()
		case scopeVersionLabel:
			scope.Version = l.GetValue()
		case scopeSchemaURLLabel:
			scope.SchemaURL = l.GetValue()
		default:
			key := strings.TrimPrefix(name, scopeLabelPrefix)
			scopeAttrs = append(scopeAttrs, attribute.String(key, l.GetValue()))
		}
	}
	if !hasScope {
		return defaultScope
	}
	if len(scopeAttrs) > 0 {
		scope.Attributes = attribute.NewSet(scopeAttrs...)
	}
	return scope
}

func convertLabels(labels []*dto.LabelPair) attribute.Set {
	kvs := make([]attribute.KeyValue, 0, len(labels))
	for _, l := range labels {
		if strings.HasPrefix(l.GetName(), scopeLabelPrefix) {
			continue
		}
		kvs = append(kvs, attribute.String(l.GetName(), l.GetValue()))
	}
	return attribute.NewSet(kvs...)
}

func convertExemplar(exemplar *dto.Exemplar) metricdata.Exemplar[float64] {
	attrs := make([]attribute.KeyValue, 0)
	var traceID, spanID []byte
	// find the trace ID and span ID in attributes, if it exists
	for _, label := range exemplar.GetLabel() {
		switch label.GetName() {
		case traceIDLabel:
			if tid, err := trace.TraceIDFromHex(label.GetValue()); err == nil {
				traceID = tid[:]
			}
		case spanIDLabel:
			if sid, err := trace.SpanIDFromHex(label.GetValue()); err == nil {
				spanID = sid[:]
			}
		default:
			attrs = append(attrs, attribute.String(label.GetName(), label.GetValue()))
		}
	}
	return metricdata.Exemplar[float64]{
		Value:              exemplar.GetValue(),
		Time:               exemplar.GetTimestamp().AsTime(),
		TraceID:            traceID,
		SpanID:             spanID,
		FilteredAttributes: attrs,
	}
}

var wordToUCUM = map[string]string{
	// Time
	"days":         "d",
	"hours":        "h",
	"minutes":      "min",
	"seconds":      "s",
	"milliseconds": "ms",
	"microseconds": "us",
	"nanoseconds":  "ns",

	// Bytes
	"bytes":     "By",
	"kibibytes": "KiBy",
	"mebibytes": "MiBy",
	"gibibytes": "GiBy",
	"tebibytes": "TiBy",
	"kilobytes": "kBy",
	"megabytes": "MBy",
	"gigabytes": "GBy",
	"terabytes": "TBy",

	// SI
	"meters":  "m",
	"volts":   "V",
	"amperes": "A",
	"joules":  "J",
	"watts":   "W",
	"grams":   "g",

	// Misc
	"celsius": "Cel",
	"hertz":   "Hz",
	"percent": "%",
}

var perWordToUCUM = map[string]string{
	"second": "s",
	"minute": "min",
	"hour":   "h",
	"day":    "d",
	"week":   "wk",
	"month":  "mo",
	"year":   "a",
}

func unitWordToUCUM(unit string) string {
	if unit == "" {
		return ""
	}
	unitTokens := strings.SplitN(unit, "_per_", 2)
	ucumUnit := wordToUCUMOrDefault(unitTokens[0])
	if len(unitTokens) > 1 && unitTokens[1] != "" {
		ucumUnit += "/" + perWordToUCUMOrDefault(unitTokens[1])
	}
	return ucumUnit
}

func wordToUCUMOrDefault(unit string) string {
	if ucum, ok := wordToUCUM[unit]; ok {
		return ucum
	}
	return unit
}

func perWordToUCUMOrDefault(perUnit string) string {
	if ucum, ok := perWordToUCUM[perUnit]; ok {
		return ucum
	}
	return perUnit
}

type multierr []error

func (e multierr) errOrNil() error {
	if len(e) == 0 {
		return nil
	} else if len(e) == 1 {
		return e[0]
	}
	return e
}

func (e multierr) Error() string {
	es := make([]string, len(e))
	for i, err := range e {
		es[i] = fmt.Sprintf("* %s", err)
	}
	return strings.Join(es, "\n\t")
}
