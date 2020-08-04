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

package push_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/notify/mock"
	"go.opentelemetry.io/contrib/sdk/dynamicconfig/metric/controller/push"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/label"
	"go.opentelemetry.io/otel/api/metric"
	exporterTest "go.opentelemetry.io/otel/exporters/metric/test"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	controllerTest "go.opentelemetry.io/otel/sdk/metric/controller/test"
	"go.opentelemetry.io/otel/sdk/metric/processor/test"
	processorTest "go.opentelemetry.io/otel/sdk/metric/processor/test"
	"go.opentelemetry.io/otel/sdk/resource"
)

var testResource = resource.New(kv.String("R", "V"))

type handler struct {
	sync.Mutex
	err error
}

func (h *handler) Handle(err error) {
	h.Lock()
	h.err = err
	h.Unlock()
}

func (h *handler) Flush() error {
	h.Lock()
	err := h.err
	h.err = nil
	h.Unlock()
	return err
}

var testHandler *handler

func init() {
	testHandler = new(handler)
	global.SetHandler(testHandler)
}

type testExporter struct {
	t         *testing.T
	lock      sync.Mutex
	exports   int
	records   []export.Record
	injectErr func(r export.Record) error
}

type testFixture struct {
	checkpointSet *exporterTest.CheckpointSet
	exporter      *testExporter
}

func newFixture(t *testing.T) testFixture {
	checkpointSet := exporterTest.NewCheckpointSet(testResource)

	exporter := &testExporter{
		t: t,
	}

	return testFixture{
		checkpointSet: checkpointSet,
		exporter:      exporter,
	}
}

func (e *testExporter) ExportKindFor(*metric.Descriptor, aggregation.Kind) export.ExportKind {
	return export.PassThroughExporter
}

func (e *testExporter) Export(_ context.Context, checkpointSet export.CheckpointSet) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.exports++
	var records []export.Record
	if err := checkpointSet.ForEach(e, func(r export.Record) error {
		if e.injectErr != nil {
			if err := e.injectErr(r); err != nil {
				return err
			}
		}
		records = append(records, r)
		return nil
	}); err != nil {
		return err
	}
	e.records = records
	return nil
}

func (e *testExporter) resetRecords() ([]export.Record, int) {
	e.lock.Lock()
	defer e.lock.Unlock()
	r := e.records
	e.records = nil
	return r, e.exports
}

func TestPushDoubleStop(t *testing.T) {
	fix := newFixture(t)
	p := push.New(processorTest.AggregatorSelector(), fix.exporter, "")
	p.SetPeriod(1)
	p.Start()
	p.Stop()
	p.Stop()
}

func TestPushDoubleStart(t *testing.T) {
	fix := newFixture(t)
	p := push.New(test.AggregatorSelector(), fix.exporter, "")
	p.SetPeriod(1)
	p.Start()
	p.Start()
	p.Stop()
}

func TestPushPeriod(t *testing.T) {
	fix := newFixture(t)

	p := push.New(
		test.AggregatorSelector(),
		fix.exporter,
		"",
		push.WithResource(testResource),
	)
	meter := p.Provider().Meter("name")

	mockClock := controllerTest.NewMockClock()
	p.SetClock(mockClock)
	p.SetPeriod(1)
	p.SetDone()

	ctx := context.Background()

	counter := metric.Must(meter).NewInt64Counter("counter.sum")

	p.Start()
	p.WaitDone()

	counter.Add(ctx, 3)

	records, exports := fix.exporter.resetRecords()
	require.Equal(t, 0, exports)
	require.Equal(t, 0, len(records))

	log.Println("[WOOT] adding time to mock timer")
	mockClock.Add(1 * time.Second)
	p.WaitDone()

	records, exports = fix.exporter.resetRecords()
	require.Equal(t, 1, exports)
	require.Equal(t, 1, len(records))
	require.Equal(t, "counter.sum", records[0].Descriptor().Name())
	require.Equal(t, "R=V", records[0].Resource().Encoded(label.DefaultEncoder()))

	sum, err := records[0].Aggregation().(aggregation.Sum).Sum()
	require.Equal(t, int64(3), sum.AsInt64())
	require.Nil(t, err)

	fix.checkpointSet.Reset()

	counter.Add(ctx, 7)

	mockClock.Add(time.Second)
	p.WaitDone()

	records, exports = fix.exporter.resetRecords()
	require.Equal(t, 2, exports)
	require.Equal(t, 1, len(records))
	require.Equal(t, "counter.sum", records[0].Descriptor().Name())
	require.Equal(t, "R=V", records[0].Resource().Encoded(label.DefaultEncoder()))

	sum, err = records[0].Aggregation().(aggregation.Sum).Sum()
	require.Equal(t, int64(7), sum.AsInt64())
	require.Nil(t, err)

	p.Stop()
	p.WaitDone()
}

func TestPushExportError(t *testing.T) {
	injector := func(name string, e error) func(r export.Record) error {
		return func(r export.Record) error {
			if r.Descriptor().Name() == name {
				return e
			}
			return nil
		}
	}
	var errAggregator = fmt.Errorf("unexpected error")
	var tests = []struct {
		name                string
		injectedError       error
		expectedDescriptors []string
		expectedError       error
	}{
		{"errNone", nil, []string{"counter1.sum{R=V,X=Y}", "counter2.sum{R=V,}"}, nil},
		{"errNoData", aggregation.ErrNoData, []string{"counter2.sum{R=V,}"}, nil},
		{"errUnexpected", errAggregator, []string{}, errAggregator},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := newFixture(t)
			fix.exporter.injectErr = injector("counter1.sum", tt.injectedError)

			p := push.New(
				test.AggregatorSelector(),
				fix.exporter,
				"",
				push.WithResource(testResource),
			)

			mockClock := controllerTest.NewMockClock()
			p.SetClock(mockClock)
			p.SetPeriod(1)
			p.SetDone()

			ctx := context.Background()

			meter := p.Provider().Meter("name")
			counter1 := metric.Must(meter).NewInt64Counter("counter1.sum")
			counter2 := metric.Must(meter).NewInt64Counter("counter2.sum")

			p.Start()
			p.WaitDone()

			counter1.Add(ctx, 3, kv.String("X", "Y"))
			counter2.Add(ctx, 5)

			require.Equal(t, 0, fix.exporter.exports)
			require.Nil(t, testHandler.Flush())

			mockClock.Add(time.Second)
			p.WaitDone()

			records, exports := fix.exporter.resetRecords()
			require.Equal(t, 1, exports)
			if tt.expectedError == nil {
				require.NoError(t, testHandler.Flush())
			} else {
				err := testHandler.Flush()
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			}
			require.Equal(t, len(tt.expectedDescriptors), len(records))
			for _, r := range records {
				require.Contains(t, tt.expectedDescriptors,
					fmt.Sprintf("%s{%s,%s}",
						r.Descriptor().Name(),
						r.Resource().Encoded(label.DefaultEncoder()),
						r.Labels().Encoded(label.DefaultEncoder()),
					),
				)
			}

			p.Stop()
			p.WaitDone()
		})
	}
}

func TestPushScheduleChange(t *testing.T) {
	oneSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "one",
				},
			},
		},
		PeriodSec: 5,
	}
	twoSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "two",
				},
			},
		},
		PeriodSec: 10,
	}
	config := pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			&oneSchedule,
			&twoSchedule,
		},
	}
	fix := newFixture(t)

	p := push.New(
		test.AggregatorSelector(),
		fix.exporter,
		"",
		push.WithResource(testResource),
	)

	mockClock := controllerTest.NewMockClock()
	p.SetClock(mockClock)

	notifier := mock.NewNotifier()
	notifier.Receive(&notify.MetricConfig{config})
	p.SetNotifier(notifier)

	p.SetDone()

	meter := p.Provider().Meter("name")
	ctx := context.Background()

	// Initially has period of 5 seconds.
	counter1 := metric.Must(meter).NewInt64Counter("one.sum")
	// Initially has period of 10 seconds.
	counter2 := metric.Must(meter).NewInt64Counter("two.sum")

	counter1.Add(ctx, 1)
	counter2.Add(ctx, 2)

	p.Start()
	p.WaitDone()

	records, _ := fix.exporter.resetRecords()
	require.Equal(t, 0, len(records))

	mockClock.Add(5 * time.Second)
	p.WaitDone()

	// After 5 seconds, expect export from counter1 instrument.
	records, _ = fix.exporter.resetRecords()
	require.Equal(t, 1, len(records))
	require.Equal(t, "one.sum", records[0].Descriptor().Name())

	fix.checkpointSet.Reset()

	counter1.Add(ctx, 1)
	mockClock.Add(5 * time.Second)
	p.WaitDone()

	// After 10 seconds, expect export from both instruments.
	// TODO: data race in test
	records, _ = fix.exporter.resetRecords()
	require.Equal(t, 2, len(records))
	require.Equal(t, "one.sum", records[0].Descriptor().Name())
	require.Equal(t, "two.sum", records[1].Descriptor().Name())

	fix.checkpointSet.Reset()

	counter1.Add(ctx, 3)
	counter2.Add(ctx, 4)

	// Update counter1's period to 10 seconds.
	oneSchedule.PeriodSec = 10
	notifier.Receive(&notify.MetricConfig{config})
	p.WaitDone()

	mockClock.Add(5 * time.Second)

	// After 5 seconds, expect no exports.
	records, _ = fix.exporter.resetRecords()
	require.Equal(t, 0, len(records))

	mockClock.Add(5 * time.Second)
	p.WaitDone()

	// After 10 seconds, expect exports from both instruments.
	records, _ = fix.exporter.resetRecords()
	require.Equal(t, 2, len(records))
	firstName := records[0].Descriptor().Name()
	secondName := records[1].Descriptor().Name()
	if "one.sum" != firstName && "one.sum" != secondName {
		t.Errorf("could not find name 'one.sum' in results: %v", []string{firstName, secondName})
	}

	if "two.sum" != firstName && "two.sum" != secondName {
		t.Errorf("could not find name 'two.sum' in results: %v", []string{firstName, secondName})
	}

	p.Stop()
	p.WaitDone()
}
