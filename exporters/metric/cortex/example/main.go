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

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func main() {
	// Create a new Config struct.
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	fmt.Println("Success: Created Config struct")

	// Create and install the exporter. Additionally, set the push interval to 5 seconds
	// and add a resource to the controller.
	cont, err := cortex.InstallNewPipeline(*config, controller.WithCollectPeriod(5*time.Second), controller.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attribute.String("R", "V"))))
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	ctx := context.Background()
	defer func() {
		handleErr(cont.Stop(ctx))
	}()
	fmt.Println("Success: Installed Exporter Pipeline")

	// Create a counter and a value recorder
	meter := cont.Meter("example")

	// Create instruments.
	histogram := metric.Must(meter).NewInt64Histogram(
		"example.histogram",
		metric.WithDescription("Records values"),
	)

	counter := metric.Must(meter).NewInt64Counter(
		"example.counter",
		metric.WithDescription("Counts things"),
	)
	fmt.Println("Success: Created Int64Histogram and Int64Counter instruments!")

	// Record random values to the instruments in a loop
	fmt.Println("Starting to write data to the instruments!")
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	for {
		time.Sleep(1 * time.Second)
		randomValue := random.Intn(100)
		value := int64(randomValue * 10)
		histogram.Record(ctx, value, attribute.String("key", "value"))
		counter.Add(ctx, int64(randomValue), attribute.String("key", "value"))
		fmt.Printf("Adding %d to counter and recording %d in histogram\n", randomValue, value)
	}

}

func handleErr(err error) {
	if err != nil {
		fmt.Println("Encountered error: ", err.Error())
	}
}
