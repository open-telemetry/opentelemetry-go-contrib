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
	"context"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd/internal/statsd"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
)

type (
	Config = statsd.Config

	// Exporter implements a dogstatsd-format statsd exporter,
	// which encodes attribute sets as independent fields in the
	// output.
	//
	// TODO: find a link for this syntax.  It's been copied out of
	// code, not a specification:
	//
	// https://github.com/stripe/veneur/blob/master/sinks/datadog/datadog.go
	Exporter struct {
		*statsd.Exporter

		attributeEncoder *AttributeEncoder
	}
)

var (
	_ export.Exporter = &Exporter{}
)

// NewRawExporter returns a new Dogstatsd-syntax exporter for use in a pipeline.
func NewRawExporter(config Config) (*Exporter, error) {
	exp := &Exporter{
		attributeEncoder: NewAttributeEncoder(),
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
func InstallNewPipeline(config Config) (*controller.Controller, error) {
	controller, err := NewExportPipeline(config, controller.WithCollectPeriod(time.Minute))
	if err != nil {
		return controller, err
	}
	global.SetMeterProvider(controller)
	return controller, err
}

// NewExportPipeline sets up a complete export pipeline with the recommended setup,
// chaining a NewRawExporter into the recommended selectors and batchers.
func NewExportPipeline(config Config, opts ...controller.Option) (*controller.Controller, error) {
	exporter, err := NewRawExporter(config)
	if err != nil {
		return nil, err
	}

	// Use arrays for Values and sums for everything else
	selector := simple.NewWithExactDistribution()

	// The basic processor ensures that the exporter sees the full
	// set of attributes as dogstatsd tags.
	processor := basic.NewFactory(selector, exporter)

	cont := controller.New(processor, append(opts, controller.WithExporter(exporter))...)

	return cont, cont.Start(context.Background())
}

// AppendName is part of the stats-internal adapter interface.
func (*Exporter) AppendName(rec export.Record, buf *bytes.Buffer) {
	_, _ = buf.WriteString(rec.Descriptor().Name())
}

// AppendTags is part of the stats-internal adapter interface.
func (e *Exporter) AppendTags(rec export.Record, res *resource.Resource, buf *bytes.Buffer) {
	rencoded := res.Encoded(e.attributeEncoder)
	lencoded := rec.Labels().Encoded(e.attributeEncoder)

	// Note: We do not de-duplicate tag-keys between resources and
	// event attributes here.  Instead, include resources first so
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
