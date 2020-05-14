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

package dogstatsd // import "go.opentelemetry.io/contrib/exporters/metric/dogstatsd"

import (
	"bytes"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd/internal/statsd"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	"go.opentelemetry.io/otel/sdk/resource"
)

type (
	Config = statsd.Config

	// Exporter implements a dogstatsd-format statsd exporter,
	// which encodes label sets as independent fields in the
	// output.
	//
	// TODO: find a link for this syntax.  It's been copied out of
	// code, not a specification:
	//
	// https://github.com/stripe/veneur/blob/master/sinks/datadog/datadog.go
	Exporter struct {
		*statsd.Exporter

		labelEncoder *LabelEncoder
	}
)

var (
	_ export.Exporter = &Exporter{}
)

// NewRawExporter returns a new Dogstatsd-syntax exporter for use in a pipeline.
func NewRawExporter(config Config) (*Exporter, error) {
	exp := &Exporter{
		labelEncoder: NewLabelEncoder(),
	}

	var err error
	exp.Exporter, err = statsd.NewExporter(config, exp)
	return exp, err
}

// InstallNewPipeline instantiates a NewExportPipeline and registers it globally.
// Typically called as:
//
// 	pipeline, err := dogstatsd.InstallNewPipeline(dogstatsd.Config{...})
// 	if err != nil {
// 		...
// 	}
// 	defer pipeline.Stop()
// 	... Done
func InstallNewPipeline(config Config) (*push.Controller, error) {
	controller, err := NewExportPipeline(config, time.Minute)
	if err != nil {
		return controller, err
	}
	global.SetMeterProvider(controller)
	return controller, err
}

// NewExportPipeline sets up a complete export pipeline with the recommended setup,
// chaining a NewRawExporter into the recommended selectors and batchers.
func NewExportPipeline(config Config, period time.Duration, opts ...push.Option) (*push.Controller, error) {
	exporter, err := NewRawExporter(config)
	if err != nil {
		return nil, err
	}

	// The simple integrator ensures that the export sees the full
	// set of labels as dogstatsd tags.
	integrator := integrator.New(exporter, false)

	pusher := push.New(integrator, exporter, period, opts...)
	pusher.Start()

	return pusher, nil
}

// AggregatorFor uses a Sum aggregator for counters, an Array
// aggregator for Measures, and a LastValue aggregator for Observers.
func (*Exporter) AggregatorFor(descriptor *metric.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case metric.ObserverKind:
		return lastvalue.New()
	case metric.MeasureKind:
		return array.New()
	default:
		return sum.New()
	}
}

// AppendName is part of the stats-internal adapter interface.
func (*Exporter) AppendName(rec export.Record, buf *bytes.Buffer) {
	_, _ = buf.WriteString(rec.Descriptor().Name())
}

// AppendTags is part of the stats-internal adapter interface.
func (e *Exporter) AppendTags(rec export.Record, res *resource.Resource, buf *bytes.Buffer) {
	rencoded := res.Encoded(e.labelEncoder)
	lencoded := rec.Labels().Encoded(e.labelEncoder)

	// Note: We do not de-duplicate tag-keys between resources and
	// event labels here.  Instead, include resources first so
	// that the receiver can apply OTel's last-value-wins
	// semantcis, if desired.
	rlen := len(rencoded)
	llen := len(lencoded)
	if rlen == 0 && llen == 0 {
		return
	}

	buf.WriteString("|#")

	_, _ = buf.WriteString(rencoded)

	if rlen != 0 && llen != 0 {
		buf.WriteRune(',')
	}

	_, _ = buf.WriteString(lencoded)
}
