package datadog_test

import (
	"context"
	"time"

	"github.com/DataDog/sketches-go/ddsketch"
	"go.opentelemetry.io/contrib/exporters/metric/datadog"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/ungrouped"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func ExampleExporter() {
	selector := simple.NewWithSketchMeasure(ddsketch.NewDefaultConfig())
	batcher := ungrouped.New(selector, export.NewDefaultLabelEncoder(), false)
	exp, err := datadog.NewExporter(datadog.Options{
		Tags: []string{"env:dev"},
	})
	if err != nil {
		panic(err)
	}
	defer exp.Close()
	pusher := push.New(batcher, exp, time.Second*10)
	defer pusher.Stop()
	pusher.Start()
	global.SetMeterProvider(pusher)
	meter := global.Meter("marwandist")
	m := metric.Must(meter).NewInt64Counter("mycounter")
	meter.RecordBatch(context.Background(), nil, m.Measurement(19))
}
