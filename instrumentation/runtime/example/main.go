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
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	stdout "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

func initMeter() *controller.Controller {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Panicf("failed to initialize metric stdout exporter %v", err)
	}
	pusher := controller.New(
		processor.New(
			simple.NewWithInexpensiveDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(time.Second*3),
	)
	pusher.Start(context.Background()) //nolint:errcheck
	global.SetMeterProvider(pusher.MeterProvider())
	return pusher
}

func main() {
	pusher := initMeter()
	defer func(pusher *controller.Controller, ctx context.Context) {
		handleErr(pusher.Stop(ctx))
	}(pusher, context.Background())

	if err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	); err != nil {
		panic(err)
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan
}

func handleErr(err error) {
	if err != nil {
		fmt.Println("Encountered error: ", err.Error())
	}
}
