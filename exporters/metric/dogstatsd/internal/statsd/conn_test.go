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

package statsd_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd/internal/statsd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/exact"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func testMeter(t *testing.T, exp export.Exporter) (context.Context, metric.Meter, *controller.Controller) {
	aggSel := testAggregatorSelector{}
	proc := processor.NewFactory(aggSel, export.CumulativeExportKindSelector())
	cont := controller.New(proc,
		controller.WithResource(testResource),
		controller.WithExporter(exp),
	)
	ctx := context.Background()

	return ctx, cont.Meter("test"), cont
}

type testAggregatorSelector struct {
}

func (testAggregatorSelector) AggregatorFor(desc *metric.Descriptor, aggPtrs ...*export.Aggregator) {
	switch {
	case strings.HasSuffix(desc.Name(), "counter"):
		aggs := sum.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	case strings.HasSuffix(desc.Name(), "gauge"):
		aggs := lastvalue.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	case strings.HasSuffix(desc.Name(), "histogram") ||
		strings.HasSuffix(desc.Name(), "timer"):
		aggs := exact.New(len(aggPtrs))
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}
	default:
		panic(fmt.Sprint("Invalid instrument name for test AggregatorSelector: ", desc.Name()))
	}
}

var testResource = resource.NewWithAttributes(semconv.SchemaURL, attribute.String("host", "value"))

// withTagsAdapter tests a dogstatsd-style statsd exporter.
type withTagsAdapter struct {
	attribute.Encoder
}

func (*withTagsAdapter) AppendName(rec export.Record, buf *bytes.Buffer) {
	_, _ = buf.WriteString(rec.Descriptor().Name())
}

func (ta *withTagsAdapter) AppendTags(rec export.Record, _ *resource.Resource, buf *bytes.Buffer) {
	_, _ = buf.WriteString("|#")
	_, _ = buf.WriteString(rec.Labels().Encoded(ta.Encoder))
}

func newWithTagsAdapter() *withTagsAdapter {
	return &withTagsAdapter{
		// Note: This uses non-statsd syntax.  (No problem.)
		attribute.DefaultEncoder(),
	}
}

// noTagsAdapter simulates a plain-statsd exporter that appends tag
// values to the metric name.
type noTagsAdapter struct {
}

func (*noTagsAdapter) AppendName(rec export.Record, buf *bytes.Buffer) {
	_, _ = buf.WriteString(rec.Descriptor().Name())

	iter := rec.Labels().Iter()
	for iter.Next() {
		tag := iter.Attribute()
		_, _ = buf.WriteString(".")
		_, _ = buf.WriteString(tag.Value.Emit())
	}
}

func (*noTagsAdapter) AppendTags(_ export.Record, _ *resource.Resource, _ *bytes.Buffer) {
}

func newNoTagsAdapter() *noTagsAdapter {
	return &noTagsAdapter{}
}

type testWriter struct {
	vec []string
}

func (w *testWriter) Write(b []byte) (int, error) {
	w.vec = append(w.vec, string(b))
	return len(b), nil
}

func TestBasicFormat(t *testing.T) {
	type adapterOutput struct {
		adapter  statsd.Adapter
		expected []string
	}

	for _, ao := range []adapterOutput{
		{
			adapter: newWithTagsAdapter(),
			expected: []string{
				"counter:%s|c|#A=B,C=D",
				"gauge:%s|g|#A=B,C=D",
				"histogram:%s|h|#A=B,C=D",
				"timer:%s|ms|#A=B,C=D",
			},
		},
		{
			adapter: newNoTagsAdapter(),
			expected: []string{
				"counter.B.D:%s|c",
				"gauge.B.D:%s|g",
				"histogram.B.D:%s|h",
				"timer.B.D:%s|ms",
			},
		},
	} {
		adapter := ao.adapter
		expected := ao.expected
		t.Run(fmt.Sprintf("%T", adapter), func(t *testing.T) {
			for _, nkind := range []number.Kind{
				number.Float64Kind,
				number.Int64Kind,
			} {
				t.Run(nkind.String(), func(t *testing.T) {
					writer := &testWriter{}
					config := statsd.Config{
						Writer:        writer,
						MaxPacketSize: 1024,
					}
					exp, err := statsd.NewExporter(config, adapter)
					if err != nil {
						t.Fatal("New error: ", err)
					}
					ctx, meter, cont := testMeter(t, exp)
					require.NoError(t, cont.Start(ctx))

					attributes := []attribute.KeyValue{
						attribute.String("A", "B"),
						attribute.String("C", "D"),
					}

					if nkind == number.Int64Kind {
						counter := metric.Must(meter).NewInt64Counter("counter")
						_ = metric.Must(meter).NewInt64GaugeObserver("gauge",
							func(_ context.Context, res metric.Int64ObserverResult) {
								res.Observe(2, attributes...)
							})
						histo := metric.Must(meter).NewInt64Histogram("histogram")
						timer := metric.Must(meter).NewInt64Histogram("timer", metric.WithUnit("ms"))
						counter.Add(ctx, 2, attributes...)
						histo.Record(ctx, 2, attributes...)
						timer.Record(ctx, 2, attributes...)
					} else {
						counter := metric.Must(meter).NewFloat64Counter("counter")
						_ = metric.Must(meter).NewFloat64GaugeObserver("gauge",
							func(_ context.Context, res metric.Float64ObserverResult) {
								res.Observe(2, attributes...)
							})
						histo := metric.Must(meter).NewFloat64Histogram("histogram")
						timer := metric.Must(meter).NewFloat64Histogram("timer", metric.WithUnit("ms"))
						counter.Add(ctx, 2, attributes...)
						histo.Record(ctx, 2, attributes...)
						timer.Record(ctx, 2, attributes...)
					}

					require.NoError(t, cont.Stop(ctx))
					require.Equal(t, 1, len(writer.vec))

					// Note we do not know the order metrics will be sent.
					wantStrings := map[string]bool{}
					haveStrings := map[string]bool{}

					for _, wt := range expected {
						wantStrings[fmt.Sprintf(wt, "2")] = true
					}
					for _, hv := range strings.Split(writer.vec[0], "\n") {
						if len(hv) != 0 {
							haveStrings[hv] = true
						}
					}

					require.EqualValues(t, wantStrings, haveStrings)

				})
			}
		})
	}
}

// Note: this test requires an reader interface with ordered delivery,
// which the SDK does not provide.  these test helpers could become
// part of "processortest" after
// https://github.com/open-telemetry/opentelemetry-go/pull/2197
// merges.

type orderedReader struct {
	// RWMutex implements locking for the `Reader` interface.
	sync.RWMutex
	ordered []libraryReader
}

type libraryReader struct {
	library instrumentation.Library
	reader  export.Reader
}

var _ export.InstrumentationLibraryReader = &orderedReader{}

func newOrderedReader() *orderedReader {
	return &orderedReader{}
}

func (p *orderedReader) add(l instrumentation.Library, r export.Reader) {
	p.ordered = append(p.ordered, libraryReader{
		library: l,
		reader:  r,
	})
}

func (p *orderedReader) ForEach(f func(library instrumentation.Library, reader export.Reader) error) error {
	for _, pair := range p.ordered {
		if err := f(pair.library, pair.reader); err != nil && !errors.Is(err, aggregation.ErrNoData) {
			return err
		}
	}
	return nil
}

func makeAttributes(offset, nkeys int) []attribute.KeyValue {
	r := make([]attribute.KeyValue, nkeys)
	for i := range r {
		r[i] = attribute.String(fmt.Sprint("k", offset+i), fmt.Sprint("v", offset+i))
	}
	return r
}

type splitTestCase struct {
	name  string
	setup func(add func(int))
	check func(expected, got []string, t *testing.T)
}

const large = 1000

var splitTestCases = []splitTestCase{
	// These test use the number of keys to control where packets
	// are split.
	{"Simple",
		func(add func(int)) {
			add(1)
			add(large)
			add(1)
		},
		func(expected, got []string, t *testing.T) {
			require.EqualValues(t, expected, got)
		},
	},
	{"LastBig",
		func(add func(int)) {
			add(1)
			add(1)
			add(large)
		},
		func(expected, got []string, t *testing.T) {
			require.Equal(t, 2, len(got))
			require.EqualValues(t, []string{
				expected[0] + expected[1],
				expected[2],
			}, got)
		},
	},
	{"FirstBig",
		func(add func(int)) {
			add(large)
			add(1)
			add(1)
			add(large)
			add(1)
			add(1)
		},
		func(expected, got []string, t *testing.T) {
			require.Equal(t, 4, len(got))
			require.EqualValues(t, []string{
				expected[0],
				expected[1] + expected[2],
				expected[3],
				expected[4] + expected[5],
			}, got)
		},
	},
	{"OneBig",
		func(add func(int)) {
			add(large)
		},
		func(expected, got []string, t *testing.T) {
			require.EqualValues(t, expected, got)
		},
	},
	{"LastSmall",
		func(add func(int)) {
			add(large)
			add(1)
		},
		func(expected, got []string, t *testing.T) {
			require.EqualValues(t, expected, got)
		},
	},
	{"Overflow",
		func(add func(int)) {
			for i := 0; i < large; i++ {
				add(1)
			}
		},
		func(expected, got []string, t *testing.T) {
			require.Less(t, 1, len(got))
			require.Equal(t, strings.Join(expected, ""), strings.Join(got, ""))
		},
	},
	{"Empty",
		func(add func(int)) {
		},
		func(expected, got []string, t *testing.T) {
			require.Equal(t, 0, len(got))
		},
	},
	{"AllBig",
		func(add func(int)) {
			add(large)
			add(large)
			add(large)
		},
		func(expected, got []string, t *testing.T) {
			require.EqualValues(t, expected, got)
		},
	},
}

func TestPacketSplit(t *testing.T) {
	for _, tcase := range splitTestCases {
		t.Run(tcase.name, func(t *testing.T) {
			ctx := context.Background()
			writer := &testWriter{}
			config := statsd.Config{
				Writer:        writer,
				MaxPacketSize: 1024,
			}
			adapter := newWithTagsAdapter()
			exp, err := statsd.NewExporter(config, adapter)
			if err != nil {
				t.Fatal("New error: ", err)
			}

			orderedReader := newOrderedReader()

			var expected []string

			offset := 0
			tcase.setup(func(nkeys int) {
				attributes := makeAttributes(offset, nkeys)
				offset += nkeys
				eattributes := attribute.NewSet(attributes...)
				encoded := adapter.Encoder.Encode(eattributes.Iter())
				expect := fmt.Sprint("counter:100|c|#", encoded, "\n")
				expected = append(expected, expect)

				ctx, meter, cont := testMeter(t, nil)

				counter := metric.Must(meter).NewInt64Counter("counter")
				counter.Add(ctx, 100, attributes...)

				require.NoError(t, cont.Collect(ctx))

				require.NoError(t, cont.ForEach(func(library instrumentation.Library, reader export.Reader) error {
					orderedReader.add(library, reader)
					return nil
				}))
			})

			err = exp.Export(ctx, testResource, orderedReader)
			require.Nil(t, err)

			tcase.check(expected, writer.vec, t)
		})
	}
}

func TestExactSplit(t *testing.T) {
	writer := &testWriter{}
	config := statsd.Config{
		Writer:        writer,
		MaxPacketSize: 1024,
	}
	adapter := newWithTagsAdapter()
	exp, err := statsd.NewExporter(config, adapter)
	if err != nil {
		t.Fatal("New error: ", err)
	}

	ctx, meter, cont := testMeter(t, exp)
	histo := metric.Must(meter).NewInt64Histogram("histogram")
	require.NoError(t, cont.Start(ctx))

	for i := 0; i < 1024; i++ {
		histo.Record(ctx, 100)
	}

	require.NoError(t, cont.Stop(ctx))

	require.Greater(t, len(writer.vec), 1)

	for _, result := range writer.vec {
		require.LessOrEqual(t, len(result), config.MaxPacketSize)
	}
}

func TestPrefix(t *testing.T) {
	writer := &testWriter{}
	config := statsd.Config{
		Writer:        writer,
		MaxPacketSize: 1024,
		Prefix:        "veryspecial.",
	}
	adapter := newWithTagsAdapter()
	exp, err := statsd.NewExporter(config, adapter)
	if err != nil {
		t.Fatal("New error: ", err)
	}

	ctx, meter, cont := testMeter(t, exp)
	histo := metric.Must(meter).NewInt64Histogram("histogram")
	require.NoError(t, cont.Start(ctx))

	histo.Record(ctx, 100)

	require.NoError(t, cont.Stop(ctx))

	require.Equal(t, `veryspecial.histogram:100|h|#
`, strings.Join(writer.vec, ""))
}
