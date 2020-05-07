package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/api/global"
	metricstdout "go.opentelemetry.io/otel/exporters/metric/stdout"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"

	"go.opentelemetry.io/contrib/plugins/metrics/runtime"
)

func initMeter() *push.Controller {
	pusher, err := metricstdout.NewExportPipeline(metricstdout.Config{
		Quantiles:   []float64{0.5},
		PrettyPrint: true,
	}, 10*time.Second)
	if err != nil {
		log.Panicf("failed to initialize metric stdout exporter %v", err)
	}
	global.SetMeterProvider(pusher)
	return pusher
}

func main() {
	defer initMeter().Stop()

	meter := global.Meter("runtime")

	r := runtime.New(meter, time.Second)
	err := r.Start()
	if err != nil {
		panic(err)
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	r.Stop()
}
