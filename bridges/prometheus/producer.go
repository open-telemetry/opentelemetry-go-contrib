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
	scopeName    = "go.opentelemetry.io/contrib/bridges/prometheus"
	traceIDLabel = "trace_id"
	spanIDLabel  = "span_id"
)

var (
	errUnsupportedType  = errors.New("unsupported metric type")
	errInvalidHistogram = errors.New("invalid histogram")
	processStartTime    = time.Now()
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
	otelMetrics := make([]metricdata.Metrics, 0)
	for _, gatherer := range p.gatherers {
		promMetrics, err := gatherer.Gather()
		if err != nil {
			errs = append(errs, err)
		}
		m, err := convertPrometheusMetricsInto(promMetrics, now)
		otelMetrics = append(otelMetrics, m...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if errs.errOrNil() != nil {
		otel.Handle(errs.errOrNil())
	}
	if len(otelMetrics) == 0 {
		return nil, nil
	}
	return []metricdata.ScopeMetrics{{
		Scope: instrumentation.Scope{
			Name: scopeName,
		},
		Metrics: otelMetrics,
	}}, nil
}

func convertPrometheusMetricsInto(promMetrics []*dto.MetricFamily, now time.Time) ([]metricdata.Metrics, error) {
	var errs multierr
	otelMetrics := make([]metricdata.Metrics, 0)
	for _, pm := range promMetrics {
		if len(pm.GetMetric()) == 0 {
			// This shouldn't ever happen
			continue
		}
		newMetric := metricdata.Metrics{
			Name:        pm.GetName(),
			Description: pm.GetHelp(),
			Unit:        unitWordToUCUM(pm.GetUnit()),
		}
		switch pm.GetType() {
		case dto.MetricType_GAUGE:
			newMetric.Data = convertGauge(pm.GetMetric(), now)
		case dto.MetricType_UNTYPED:
			newMetric.Data = convertUntyped(pm.GetMetric(), now)
		case dto.MetricType_COUNTER:
			newMetric.Data = convertCounter(pm.GetMetric(), now)
		case dto.MetricType_SUMMARY:
			newMetric.Data = convertSummary(pm.GetMetric(), now)
		case dto.MetricType_HISTOGRAM:
			var validMetrics []*dto.Metric
			for _, m := range pm.GetMetric() {
				if err := validateHistogram(pm.GetName(), m.GetHistogram()); err != nil {
					errs = append(errs, err)
					continue
				}
				validMetrics = append(validMetrics, m)
			}
			if len(validMetrics) == 0 {
				continue
			}
			if isExponentialHistogram(validMetrics[0].GetHistogram()) {
				data, err := convertExponentialHistogram(pm.GetName(), validMetrics, now)
				if err != nil {
					errs = append(errs, err)
				}
				if len(data.DataPoints) == 0 {
					continue
				}
				newMetric.Data = data
			} else {
				data, err := convertHistogram(pm.GetName(), validMetrics, now)
				if err != nil {
					errs = append(errs, err)
				}
				if len(data.DataPoints) == 0 {
					continue
				}
				newMetric.Data = data
			}
		default:
			// MetricType_GAUGE_HISTOGRAM
			errs = append(errs, fmt.Errorf("%w: %v for metric %v", errUnsupportedType, pm.GetType(), pm.GetName()))
			continue
		}
		otelMetrics = append(otelMetrics, newMetric)
	}
	return otelMetrics, errs.errOrNil()
}

func validateHistogram(metricName string, hist *dto.Histogram) error {
	// The OpenTelemetry Prometheus and OpenMetrics Compatibility Specification requires:
	// "Native histograms of the float or gauge flavors MUST be dropped."
	if isFloatHistogram(hist) {
		return fmt.Errorf("%w: float histogram for metric %v", errUnsupportedType, metricName)
	}
	if schema := hist.GetSchema(); schema < -4 || schema > 8 {
		return fmt.Errorf("%w: schema %d for metric %v", errUnsupportedType, schema, metricName)
	}
	return nil
}

func isExponentialHistogram(hist *dto.Histogram) bool {
	// The prometheus go client ensures at least one of these is non-zero
	// so it can be distinguished from a fixed-bucket histogram.
	// https://github.com/prometheus/client_golang/blob/7ac90362b02729a65109b33d172bafb65d7dab50/prometheus/histogram.go#L818
	return hist.GetZeroThreshold() > 0 ||
		hist.GetZeroCount() > 0 ||
		len(hist.GetPositiveSpan()) > 0 ||
		len(hist.GetNegativeSpan()) > 0 ||
		isFloatHistogram(hist)
}

func isFloatHistogram(hist *dto.Histogram) bool {
	return hist.GetSampleCountFloat() > 0 ||
		hist.GetZeroCountFloat() > 0 ||
		len(hist.GetPositiveCount()) > 0 ||
		len(hist.GetNegativeCount()) > 0
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

func convertExponentialHistogram(metricName string, metrics []*dto.Metric, now time.Time) (metricdata.ExponentialHistogram[float64], error) {
	var errs multierr
	otelExpHistogram := metricdata.ExponentialHistogram[float64]{
		DataPoints:  make([]metricdata.ExponentialHistogramDataPoint[float64], 0, len(metrics)),
		Temporality: metricdata.CumulativeTemporality,
	}
	for _, m := range metrics {
		hist := m.GetHistogram()
		if err := validateHistogram(metricName, hist); err != nil {
			errs = append(errs, err)
			continue
		}
		// The OpenTelemetry Prometheus and OpenMetrics Compatibility Specification requires:
		// "A Prometheus Native Histogram with a standard exponential schema (schemas -4 to 8)
		// of the integer and counter flavor MUST be converted to an OTLP Exponential Histogram."
		// Non-matching classic histogram datapoints in an exponential family must be rejected.
		if !isExponentialHistogram(hist) {
			errs = append(errs, fmt.Errorf("%w: missing exponential histogram buckets for metric %v", errInvalidHistogram, metricName))
			continue
		}
		positiveBucket, err := convertExponentialBuckets(
			hist.GetPositiveSpan(),
			hist.GetPositiveDelta(),
		)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w for metric %v", err, metricName))
			continue
		}
		negativeBucket, err := convertExponentialBuckets(
			hist.GetNegativeSpan(),
			hist.GetNegativeDelta(),
		)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w for metric %v", err, metricName))
			continue
		}
		dp := metricdata.ExponentialHistogramDataPoint[float64]{
			Attributes:     convertLabels(m.GetLabel()),
			StartTime:      processStartTime,
			Time:           now,
			Count:          hist.GetSampleCount(),
			Sum:            hist.GetSampleSum(),
			Scale:          hist.GetSchema(),
			ZeroCount:      hist.GetZeroCount(),
			ZeroThreshold:  hist.GetZeroThreshold(),
			PositiveBucket: positiveBucket,
			NegativeBucket: negativeBucket,
		}
		if exs := m.GetHistogram().GetExemplars(); len(exs) > 0 {
			dp.Exemplars = make([]metricdata.Exemplar[float64], len(exs))
			for j, ex := range exs {
				dp.Exemplars[j] = convertExemplar(ex)
			}
		}
		createdTs := hist.GetCreatedTimestamp()
		if createdTs.IsValid() {
			dp.StartTime = createdTs.AsTime()
		}
		if m.TimestampMs != nil {
			dp.Time = time.UnixMilli(m.GetTimestampMs())
		}
		otelExpHistogram.DataPoints = append(otelExpHistogram.DataPoints, dp)
	}
	return otelExpHistogram, errs.errOrNil()
}

func convertExponentialBuckets(bucketSpans []*dto.BucketSpan, deltas []int64) (metricdata.ExponentialBucket, error) {
	// We will have one bucket count for each delta, and zeros for the offsets
	// after the initial offset.
	lenCounts := len(deltas)
	totalLength := 0
	for i, bs := range bucketSpans {
		if i != 0 {
			if bs.GetOffset() < 0 {
				return metricdata.ExponentialBucket{}, fmt.Errorf("%w: negative bucket span offset %d", errInvalidHistogram, bs.GetOffset())
			}
			lenCounts += int(bs.GetOffset())
		}
		if uint64(bs.GetLength()) > uint64(len(deltas)-totalLength) {
			return metricdata.ExponentialBucket{}, fmt.Errorf("%w: bucket span lengths exceed deltas length %d", errInvalidHistogram, len(deltas))
		}
		totalLength += int(bs.GetLength())
	}
	if totalLength != len(deltas) {
		return metricdata.ExponentialBucket{}, fmt.Errorf("%w: bucket span lengths (%d) do not match deltas length (%d)", errInvalidHistogram, totalLength, len(deltas))
	}
	if totalLength == 0 {
		return metricdata.ExponentialBucket{}, nil
	}
	// Prometheus Native Histograms buckets are indexed by upper boundary
	// while Exponential Histograms are indexed by lower boundary, the result
	// being that the Offset fields are different-by-one.
	initialOffset := bucketSpans[0].GetOffset() - 1
	counts := make([]uint64, lenCounts)
	deltaIndex := 0
	countIndex := 0
	count := int64(0)
	for i, bs := range bucketSpans {
		// Do not insert zeroes if this is the first bucketSpan, since those
		// zeroes are accounted for in the Offset field.
		if i != 0 {
			// Increase the count index by the Offset to insert Offset zeroes
			countIndex += int(bs.GetOffset())
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
	}, nil
}

func convertHistogram(metricName string, metrics []*dto.Metric, now time.Time) (metricdata.Histogram[float64], error) {
	var errs multierr
	otelHistogram := metricdata.Histogram[float64]{
		DataPoints:  make([]metricdata.HistogramDataPoint[float64], 0, len(metrics)),
		Temporality: metricdata.CumulativeTemporality,
	}
	for _, m := range metrics {
		// The OpenTelemetry Prometheus and OpenMetrics Compatibility Specification requires:
		// "Traditional Prometheus Histograms (with explicit bucket boundaries) MUST be converted to an OTLP Histogram."
		// Non-matching exponential histogram datapoints lacking explicit buckets must be rejected.
		if isExponentialHistogram(m.GetHistogram()) && len(m.GetHistogram().GetBucket()) == 0 {
			errs = append(errs, fmt.Errorf("%w: missing histogram buckets for metric %v", errInvalidHistogram, metricName))
			continue
		}
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
		otelHistogram.DataPoints = append(otelHistogram.DataPoints, dp)
	}
	return otelHistogram, errs.errOrNil()
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

func convertLabels(labels []*dto.LabelPair) attribute.Set {
	kvs := make([]attribute.KeyValue, len(labels))
	for i, l := range labels {
		kvs[i] = attribute.String(l.GetName(), l.GetValue())
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

func (e multierr) Unwrap() []error {
	return e
}
