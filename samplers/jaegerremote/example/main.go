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
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"

	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	jaegerRemoteSampler := jaegerremote.New(
		"foo",
		jaegerremote.WithSamplingServerURL("http://localhost:5778"),
		jaegerremote.WithSamplingRefreshInterval(10*time.Second), // decrease polling interval to get quicker feedback
		jaegerremote.WithInitialSampler(trace.TraceIDRatioBased(0.5)),
	)

	exporter, _ := stdouttrace.New()

	tp := trace.NewTracerProvider(
		trace.WithSampler(jaegerRemoteSampler),
		trace.WithSyncer(exporter), // for production usage, use trace.WithBatcher(exporter)
	)
	otel.SetTracerProvider(tp)

	ticker := time.Tick(time.Second)
	for {
		<-ticker
		fmt.Printf("\n* Jaeger Remote Sampler %v\n\n", time.Now())
		spewCfg := spew.ConfigState{
			Indent:                  "    ",
			DisablePointerAddresses: true,
		}
		spewCfg.Dump(jaegerRemoteSampler)
	}
}
