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

	"go.opentelemetry.io/otel/exporters/stdout"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

func initMeter() *controller.Controller {
	pusher, err := stdout.InstallNewPipeline([]stdout.Option{
		stdout.WithPrettyPrint(),
	}, nil)
	if err != nil {
		log.Panicf("failed to initialize metric stdout exporter %v", err)
	}
	return pusher
}

func main() {
	defer handleErr(initMeter().Stop(context.Background()))

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
