package datadog_test

import (
	"context"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/DataDog/sketches-go/ddsketch"

	"go.opentelemetry.io/contrib/exporters/metric/datadog"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func ExampleExporter() {
	selector := simple.NewWithSketchDistribution(ddsketch.NewDefaultConfig())
	integrator := integrator.New(selector, false)
	exp, err := datadog.NewExporter(datadog.Options{
		Tags:          []string{"env:dev"},
		StatsDOptions: []statsd.Option{statsd.WithoutTelemetry()},
	})
	if err != nil {
		panic(err)
	}
	defer exp.Close()
	pusher := push.New(integrator, exp, push.WithPeriod(time.Second*10))
	defer pusher.Stop()
	pusher.Start()
	global.SetMeterProvider(pusher.Provider())
	meter := global.Meter("marwandist")
	m := metric.Must(meter).NewInt64Counter("mycounter")
	meter.RecordBatch(context.Background(), nil, m.Measurement(19))
}
