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
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	exporter, _ := stdouttrace.New(
		stdouttrace.WithoutTimestamps(),
	)

	jaegerRemoteSampler := jaegerremote.New(
		// decrease polling interval to get quicker feedback
		jaegerremote.WithPollingInterval(10*time.Second),
		// once the strategy is fetched, sample rate will drop
		jaegerremote.WithInitialSamplingRate(1),
	)

	tp := trace.NewTracerProvider(
		trace.WithSampler(jaegerRemoteSampler),
		trace.WithSyncer(exporter), // for production usage, use trace.WithBatcher(exp)
	)
	otel.SetTracerProvider(tp)

	go generateSpans()

	// wait until program is interrupted
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func generateSpans() {
	tracer := otel.GetTracerProvider().Tracer("example")

	for {
		_, span := tracer.Start(context.Background(), "span created at "+time.Now().String())
		time.Sleep(100 * time.Millisecond)
		span.End()

		time.Sleep(900 * time.Millisecond)
	}
}
