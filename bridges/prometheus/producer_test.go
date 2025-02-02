// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus // import "go.opentelemetry.io/contrib/bridges/prometheus"

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

const (
	traceIDStr = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanIDStr  = "00f067aa0ba902b7"
)

func TestProduce(t *testing.T) {
	testCases := []struct {
		name     string
		testFn   func(*prometheus.Registry)
		expected []metricdata.ScopeMetrics
		wantErr  error
	}{
		{
			name:   "no metrics registered",
			testFn: func(*prometheus.Registry) {},
		},
		{
			name: "gauge",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewGauge(prometheus.GaugeOpts{
					Name: "test_gauge_metric",
					Help: "A gauge metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Set(123.4)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_gauge_metric",
						Description: "A gauge metric for testing",
						Data: metricdata.Gauge[float64]{
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Attributes: attribute.NewSet(attribute.String("foo", "bar")),
									Value:      123.4,
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "counter",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "test_counter_metric",
					Help: "A counter metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Add(245.3)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_counter_metric",
						Description: "A counter metric for testing",
						Data: metricdata.Sum[float64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Attributes: attribute.NewSet(attribute.String("foo", "bar")),
									Value:      245.3,
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "counter with exemplar",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "test_counter_metric",
					Help: "A counter metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.(prometheus.ExemplarAdder).AddWithExemplar(
					245.3, prometheus.Labels{
						"trace_id":        traceIDStr,
						"span_id":         spanIDStr,
						"other_attribute": "abcd",
					},
				)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_counter_metric",
						Description: "A counter metric for testing",
						Data: metricdata.Sum[float64]{
							Temporality: metricdata.CumulativeTemporality,
							IsMonotonic: true,
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Attributes: attribute.NewSet(attribute.String("foo", "bar")),
									Value:      245.3,
									Exemplars: []metricdata.Exemplar[float64]{
										{
											Value:              245.3,
											TraceID:            []byte(traceIDStr),
											SpanID:             []byte(spanIDStr),
											FilteredAttributes: []attribute.KeyValue{attribute.String("other_attribute", "abcd")},
										},
									},
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "summary",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewSummary(prometheus.SummaryOpts{
					Name:       "test_summary_metric",
					Help:       "A summary metric for testing",
					Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(15.0)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_summary_metric",
						Description: "A summary metric for testing",
						Data: metricdata.Summary{
							DataPoints: []metricdata.SummaryDataPoint{
								{
									Count: 1,
									Sum:   15.0,
									QuantileValues: []metricdata.QuantileValue{
										{Quantile: 0.5, Value: 15},
										{Quantile: 0.9, Value: 15},
										{Quantile: 0.99, Value: 15},
									},
									Attributes: attribute.NewSet(attribute.String("foo", "bar")),
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "histogram",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_histogram_metric",
					Help: "A histogram metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(578.3)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_histogram_metric",
						Description: "A histogram metric for testing",
						Data: metricdata.Histogram[float64]{
							Temporality: metricdata.CumulativeTemporality,
							DataPoints: []metricdata.HistogramDataPoint[float64]{
								{
									Count:        1,
									Sum:          578.3,
									Bounds:       []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
									BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									Attributes:   attribute.NewSet(attribute.String("foo", "bar")),
									Exemplars:    []metricdata.Exemplar[float64]{},
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "histogram cumulative values to non-cumulative",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_histogram_metric",
					Help: "A histogram metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(0.01)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_histogram_metric",
						Description: "A histogram metric for testing",
						Data: metricdata.Histogram[float64]{
							Temporality: metricdata.CumulativeTemporality,
							DataPoints: []metricdata.HistogramDataPoint[float64]{
								{
									Count:        1,
									Sum:          0.01,
									Bounds:       []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
									BucketCounts: []uint64{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
									Attributes:   attribute.NewSet(attribute.String("foo", "bar")),
									Exemplars:    []metricdata.Exemplar[float64]{},
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "histogram with exemplar",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_histogram_metric_with_exemplar",
					Help: "A histogram metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.(prometheus.ExemplarObserver).ObserveWithExemplar(
					578.3, prometheus.Labels{
						"trace_id":        traceIDStr,
						"span_id":         spanIDStr,
						"other_attribute": "efgh",
					},
				)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_histogram_metric_with_exemplar",
						Description: "A histogram metric for testing",
						Data: metricdata.Histogram[float64]{
							Temporality: metricdata.CumulativeTemporality,
							DataPoints: []metricdata.HistogramDataPoint[float64]{
								{
									Count:        1,
									Sum:          578.3,
									Bounds:       []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
									BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									Attributes:   attribute.NewSet(attribute.String("foo", "bar")),
									Exemplars: []metricdata.Exemplar[float64]{
										{
											Value:   578.3,
											TraceID: []byte(traceIDStr),
											SpanID:  []byte(spanIDStr),
											FilteredAttributes: []attribute.KeyValue{
												attribute.String("other_attribute", "efgh"),
											},
										},
									},
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "exponential histogram",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_exponential_histogram_metric",
					Help: "An exponential histogram metric for testing",
					// This enables collection of native histograms in the prometheus client.
					NativeHistogramBucketFactor: 1.5,
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(78.3)
				metric.Observe(2.3)
				metric.Observe(2.3)
				metric.Observe(.5)
				metric.Observe(-78.3)
				metric.Observe(-.15)
				metric.Observe(0.0)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_exponential_histogram_metric",
						Description: "An exponential histogram metric for testing",
						Data: metricdata.ExponentialHistogram[float64]{
							Temporality: metricdata.CumulativeTemporality,
							DataPoints: []metricdata.ExponentialHistogramDataPoint[float64]{
								{
									Count:     7,
									Sum:       4.949999999999994,
									Scale:     1,
									ZeroCount: 1,
									PositiveBucket: metricdata.ExponentialBucket{
										Offset: -3,
										Counts: []uint64{1, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									},
									NegativeBucket: metricdata.ExponentialBucket{
										Offset: -6,
										Counts: []uint64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									},
									Attributes:    attribute.NewSet(attribute.String("foo", "bar")),
									ZeroThreshold: prometheus.DefNativeHistogramZeroThreshold,
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "exponential histogram with only positive observations",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_exponential_histogram_metric",
					Help: "An exponential histogram metric for testing",
					// This enables collection of native histograms in the prometheus client.
					NativeHistogramBucketFactor: 1.5,
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(78.3)
				metric.Observe(2.3)
				metric.Observe(2.3)
				metric.Observe(.5)
				metric.Observe(0.0)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_exponential_histogram_metric",
						Description: "An exponential histogram metric for testing",
						Data: metricdata.ExponentialHistogram[float64]{
							Temporality: metricdata.CumulativeTemporality,
							DataPoints: []metricdata.ExponentialHistogramDataPoint[float64]{
								{
									Count:     5,
									Sum:       83.39999999999999,
									Scale:     1,
									ZeroCount: 1,
									PositiveBucket: metricdata.ExponentialBucket{
										Offset: -3,
										Counts: []uint64{1, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
									},
									NegativeBucket: metricdata.ExponentialBucket{},
									Attributes:     attribute.NewSet(attribute.String("foo", "bar")),
									ZeroThreshold:  prometheus.DefNativeHistogramZeroThreshold,
								},
							},
						},
					},
				},
			}},
		},
		{
			name: "partial success",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewGauge(prometheus.GaugeOpts{
					Name: "test_gauge_metric",
					Help: "A gauge metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Set(123.4)
				unsupportedMetric := prometheus.NewUntypedFunc(prometheus.UntypedOpts{
					Name: "test_untyped_metric",
					Help: "An untyped metric for testing",
				}, func() float64 {
					return 135.8
				})
				reg.MustRegister(unsupportedMetric)
			},
			expected: []metricdata.ScopeMetrics{{
				Scope: instrumentation.Scope{
					Name: scopeName,
				},
				Metrics: []metricdata.Metrics{
					{
						Name:        "test_gauge_metric",
						Description: "A gauge metric for testing",
						Data: metricdata.Gauge[float64]{
							DataPoints: []metricdata.DataPoint[float64]{
								{
									Attributes: attribute.NewSet(attribute.String("foo", "bar")),
									Value:      123.4,
								},
							},
						},
					},
				},
			}},
			wantErr: errUnsupportedType,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			tt.testFn(reg)
			p := NewMetricProducer(WithGatherer(reg))
			output, err := p.Produce(context.Background())
			if tt.wantErr == nil {
				assert.NoError(t, err)
			}
			require.Equal(t, len(output), len(tt.expected))
			for i := range output {
				metricdatatest.AssertEqual(t, tt.expected[i], output[i], metricdatatest.IgnoreTimestamp())
			}
		})
	}
}

func TestProduceForStartTime(t *testing.T) {
	testCases := []struct {
		name        string
		testFn      func(*prometheus.Registry)
		startTimeFn func(metricdata.Aggregation) []time.Time
	}{
		{
			name: "counter",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "test_counter_metric",
					Help: "A counter metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.(prometheus.ExemplarAdder).AddWithExemplar(
					245.3, prometheus.Labels{
						"trace_id":        traceIDStr,
						"span_id":         spanIDStr,
						"other_attribute": "abcd",
					},
				)
			},
			startTimeFn: func(aggr metricdata.Aggregation) []time.Time {
				dps := aggr.(metricdata.Sum[float64]).DataPoints
				sts := make([]time.Time, len(dps))
				for i, dp := range dps {
					sts[i] = dp.StartTime
				}
				return sts
			},
		},
		{
			name: "histogram",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_histogram_metric",
					Help: "A histogram metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.(prometheus.ExemplarObserver).ObserveWithExemplar(
					578.3, prometheus.Labels{
						"trace_id":        traceIDStr,
						"span_id":         spanIDStr,
						"other_attribute": "efgh",
					},
				)
			},
			startTimeFn: func(aggr metricdata.Aggregation) []time.Time {
				dps := aggr.(metricdata.Histogram[float64]).DataPoints
				sts := make([]time.Time, len(dps))
				for i, dp := range dps {
					sts[i] = dp.StartTime
				}
				return sts
			},
		},
		{
			name: "summary",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewSummary(prometheus.SummaryOpts{
					Name: "test_summary_metric",
					Help: "A summary metric for testing",
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(78.3)
			},
			startTimeFn: func(aggr metricdata.Aggregation) []time.Time {
				dps := aggr.(metricdata.Summary).DataPoints
				sts := make([]time.Time, len(dps))
				for i, dp := range dps {
					sts[i] = dp.StartTime
				}
				return sts
			},
		},
		{
			name: "exponential histogram",
			testFn: func(reg *prometheus.Registry) {
				metric := prometheus.NewHistogram(prometheus.HistogramOpts{
					Name: "test_exponential_histogram_metric",
					Help: "An exponential histogram metric for testing",
					// This enables collection of native histograms in the prometheus client.
					NativeHistogramBucketFactor: 1.5,
					ConstLabels: prometheus.Labels(map[string]string{
						"foo": "bar",
					}),
				})
				reg.MustRegister(metric)
				metric.Observe(78.3)
			},
			startTimeFn: func(aggr metricdata.Aggregation) []time.Time {
				dps := aggr.(metricdata.ExponentialHistogram[float64]).DataPoints
				sts := make([]time.Time, len(dps))
				for i, dp := range dps {
					sts[i] = dp.StartTime
				}
				return sts
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			tt.testFn(reg)
			p := NewMetricProducer(WithGatherer(reg))
			output, err := p.Produce(context.Background())
			assert.NoError(t, err)
			assert.NotEmpty(t, output)
			for _, sms := range output {
				assert.NotEmpty(t, sms.Metrics)
				for _, ms := range sms.Metrics {
					sts := tt.startTimeFn(ms.Data)
					assert.NotEmpty(t, sts)
					for _, st := range sts {
						assert.True(t, st.After(processStartTime))
					}
				}
			}
		})
	}
}
