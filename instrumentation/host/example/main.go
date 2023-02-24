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

//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var res = resource.NewWithAttributes(
	semconv.SchemaURL,
	semconv.ServiceName("host-instrumentation-example"),
)

func main() {
	exp, err := stdoutmetric.New()
	if err != nil {
		log.Fatal(err)
	}

	// Register the exporter with an SDK via a periodic reader.
	read := metric.NewPeriodicReader(exp, metric.WithInterval(1*time.Second))
	provider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(read))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Print("Starting host instrumentation:")
	err = host.Start(host.WithMeterProvider(provider))
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}
