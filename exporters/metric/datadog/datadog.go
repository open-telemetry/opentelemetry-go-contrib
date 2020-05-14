package datadog

import (
	"context"
	"fmt"
	"regexp"

	"github.com/DataDog/datadog-go/statsd"

	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	// DefaultStatsAddrUDP specifies the default protocol (UDP) and address
	// for the DogStatsD service.
	DefaultStatsAddrUDP = "localhost:8125"
)

// NewExporter exports to a datadog client
func NewExporter(opts Options) (*Exporter, error) {
	if opts.StatsAddr == "" {
		opts.StatsAddr = DefaultStatsAddrUDP
	}
	if opts.MetricNameFormatter == nil {
		opts.MetricNameFormatter = defaultFormatter
	}
	client, err := statsd.New(opts.StatsAddr)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		client: client,
		opts:   opts,
	}, nil
}

// Options contains options for configuring the exporter.
type Options struct {
	// StatsAddr specifies the host[:port] address for DogStatsD. It defaults
	// to localhost:8125.
	StatsAddr string

	// Tags specifies a set of global tags to attach to each metric.
	Tags []string

	// UseDistribution uses a DataDog Distribution type instead of Histogram
	UseDistribution bool

	// MetricNameFormatter lets you customize the metric name that gets sent to
	// datadog before exporting
	MetricNameFormatter func(namespace, name string) string
}

// Exporter forwards metrics to a DataDog agent
type Exporter struct {
	opts   Options
	client *statsd.Client
}

const rate = 1

func defaultFormatter(namespace, name string) string {
	return name
}

func (e *Exporter) Export(ctx context.Context, _ *resource.Resource, cs export.CheckpointSet) error {
	// TODO: Use the Resource argument.
	return cs.ForEach(func(r export.Record) error {
		agg := r.Aggregator()
		name := e.sanitizeMetricName(r.Descriptor().LibraryName(), r.Descriptor().Name())
		itr := r.Labels().Iter()
		tags := append([]string{}, e.opts.Tags...)
		for itr.Next() {
			label := itr.Label()
			tag := string(label.Key) + ":" + label.Value.Emit()
			tags = append(tags, tag)
		}
		switch agg := agg.(type) {
		case aggregator.Points:
			numbers, err := agg.Points()
			if err != nil {
				return fmt.Errorf("error getting Points for %s: %w", name, err)
			}
			f := e.client.Histogram
			if e.opts.UseDistribution {
				f = e.client.Distribution
			}
			for _, n := range numbers {
				if err := f(name, metricValue(r.Descriptor().NumberKind(), n), tags, rate); err != nil {
					return fmt.Errorf("error submitting %s point: %w", name, err)
				}
			}
		case aggregator.MinMaxSumCount:
			type record struct {
				name string
				f    func() (metric.Number, error)
			}
			recs := []record{
				{
					name: name + ".min",
					f:    agg.Min,
				},
				{
					name: name + ".max",
					f:    agg.Max,
				},
			}
			if dist, ok := agg.(aggregator.Distribution); ok {
				recs = append(recs,
					record{name: name + ".median", f: func() (metric.Number, error) {
						return dist.Quantile(0.5)
					}},
					record{name: name + ".p95", f: func() (metric.Number, error) {
						return dist.Quantile(0.95)
					}},
				)
			}
			for _, rec := range recs {
				val, err := rec.f()
				if err != nil {
					return fmt.Errorf("error getting MinMaxSumCount value for %s: %w", name, err)
				}
				if err := e.client.Gauge(rec.name, metricValue(r.Descriptor().NumberKind(), val), tags, rate); err != nil {
					return fmt.Errorf("error submitting %s point: %w", name, err)
				}
			}
		case aggregator.Sum:
			val, err := agg.Sum()
			if err != nil {
				return fmt.Errorf("error getting Sum value for %s: %w", name, err)
			}
			if err := e.client.Count(name, val.AsInt64(), tags, rate); err != nil {
				return fmt.Errorf("error submitting %s point: %w", name, err)
			}
		case aggregator.LastValue:
			val, _, err := agg.LastValue()
			if err != nil {
				return fmt.Errorf("error getting LastValue for %s: %w", name, err)
			}
			if err := e.client.Gauge(name, metricValue(r.Descriptor().NumberKind(), val), tags, rate); err != nil {
				return fmt.Errorf("error submitting %s point: %w", name, err)
			}
		}
		return nil
	})
}

// Close cloess the underlying datadog client which flushes
// any pending buffers
func (e *Exporter) Close() error {
	return e.client.Close()
}

// sanitizeMetricName formats the custom namespace and view name to
// Datadog's metric naming convention
func (e *Exporter) sanitizeMetricName(namespace, name string) string {
	return sanitizeString(e.opts.MetricNameFormatter(namespace, name))
}

// regex pattern
var reg = regexp.MustCompile("[^a-zA-Z0-9]+")

// sanitizeString replaces all non-alphanumerical characters to underscore
func sanitizeString(str string) string {
	return reg.ReplaceAllString(str, "_")
}

func metricValue(kind metric.NumberKind, number metric.Number) float64 {
	switch kind {
	case metric.Float64NumberKind:
		return number.AsFloat64()
	case metric.Int64NumberKind:
		return float64(number.AsInt64())
	case metric.Uint64NumberKind:
		return float64(number.AsUint64())
	}
	return float64(number)
}
