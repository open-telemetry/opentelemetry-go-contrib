package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func main() {
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	fmt.Println("Success: Created Config struct")

	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(5*time.Second), push.WithResource(resource.New(kv.String("R", "V"))))
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	defer pusher.Stop()
	fmt.Println("Success: Installed Exporter Pipeline")

	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	recorder := metric.Must(meter).NewInt64ValueRecorder(
		"pipeline.valuerecorder",
		metric.WithDescription("Records values"),
	)

	counter := metric.Must(meter).NewInt64Counter(
		"pipeline.counter",
		metric.WithDescription("Counts things"),
	)
	fmt.Println("Success: Created Int64ValueRecorder and Int64Counter instruments")

	fmt.Println("Starting to write data to the instruments")
	for i := 1; i <= 10; i++ {
		time.Sleep(5 * time.Second)
		value := int64(i * 100)
		recorder.Record(ctx, value, kv.String("key", "value"))
		counter.Add(ctx, int64(i), kv.String("key", "value"))
		fmt.Printf("%d. Adding %d to counter and recording %d in recorder\n", i, i, value)
	}

}
