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

package datadog

import (
	"context"
	"fmt"
	"regexp"

	"github.com/DataDog/datadog-go/statsd"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
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
	client, err := statsd.New(opts.StatsAddr, opts.StatsDOptions...)
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
	// to DefaultStatsAddrUDP.
	StatsAddr string

	// Tags specifies a set of global tags to attach to each metric.
	Tags []string

	// UseDistribution uses a DataDog Distribution type instead of Histogram
	UseDistribution bool

	// MetricNameFormatter lets you customize the metric name that gets sent to
	// datadog before exporting
	MetricNameFormatter func(namespace, name string) string

	// StatsD specific Options
	StatsDOptions []statsd.Option
}

// Exporter forwards metrics to a DataDog agent
type Exporter struct {
	opts   Options
	client *statsd.Client
}

var (
	_ export.Exporter = &Exporter{}
)

const rate = 1

func defaultFormatter(namespace, name string) string {
	return name
}

// ExportKindFor returns export.DeltaExporter for statsd-derived exporters
func (e *Exporter) ExportKindFor(*metric.Descriptor, aggregation.Kind) export.ExportKind {
	return export.DeltaExportKind
}

func (e *Exporter) Export(ctx context.Context, res *resource.Resource, ilr export.InstrumentationLibraryReader) error {
	return ilr.ForEach(func(library instrumentation.Library, reader export.Reader) error {
		return reader.ForEach(e, func(r export.Record) error {
			// TODO: Use the Resource() method
			agg := r.Aggregation()
			name := e.sanitizeMetricName(library.Name, r.Descriptor().Name())
			itr := attribute.NewMergeIterator(r.Labels(), res.Set())
			tags := append([]string{}, e.opts.Tags...)
			for itr.Next() {
				attribute := itr.Label()
				tag := string(attribute.Key) + ":" + attribute.Value.Emit()
				tags = append(tags, tag)
			}
			switch agg := agg.(type) {
			case aggregation.Points:
				numbers, err := agg.Points()
				if err != nil {
					return fmt.Errorf("error getting Points for %s: %w", name, err)
				}
				f := e.client.Histogram
				if e.opts.UseDistribution {
					f = e.client.Distribution
				}
				for _, n := range numbers {
					if err := f(name, metricValue(r.Descriptor().NumberKind(), n.Number), tags, rate); err != nil {
						return fmt.Errorf("error submitting %s point: %w", name, err)
					}
				}
			case aggregation.MinMaxSumCount:
				type record struct {
					name string
					f    func() (number.Number, error)
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
				for _, rec := range recs {
					val, err := rec.f()
					if err != nil {
						return fmt.Errorf("error getting MinMaxSumCount value for %s: %w", name, err)
					}
					if err := e.client.Gauge(rec.name, metricValue(r.Descriptor().NumberKind(), val), tags, rate); err != nil {
						return fmt.Errorf("error submitting %s point: %w", name, err)
					}
				}
			case aggregation.Sum:
				val, err := agg.Sum()
				if err != nil {
					return fmt.Errorf("error getting Sum value for %s: %w", name, err)
				}
				if err := e.client.Count(name, val.AsInt64(), tags, rate); err != nil {
					return fmt.Errorf("error submitting %s point: %w", name, err)
				}
			case aggregation.LastValue:
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

func metricValue(kind number.Kind, num number.Number) float64 {
	switch kind {
	case number.Float64Kind:
		return num.AsFloat64()
	case number.Int64Kind:
		return float64(num.AsInt64())
	}
	return float64(num)
}
