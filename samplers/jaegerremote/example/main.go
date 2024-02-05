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
	stdlog "log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-logr/stdr"

	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Optional: an implementation of logr.Logger used for demo purposes to catch potential error logs
	logger := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})

	samplingRefreshInterval := 1 * time.Minute
	jaegerRemoteSampler := jaegerremote.New(
		"foo",
		jaegerremote.WithSamplingServerURL("http://localhost:5778/sampling"),
		jaegerremote.WithSamplingRefreshInterval(samplingRefreshInterval), // decrease polling interval to get quicker feedback
		jaegerremote.WithInitialSampler(trace.TraceIDRatioBased(0.5)),
		jaegerremote.WithLogger(logger),
	)
	// Optional: you can decorate the jaeger sampler with parent based sampler as you wish
	// parentBasedJaegerRemoteSampler := trace.ParentBased(jaegerRemoteSampler, trace.WithRemoteParentNotSampled(jaegerRemoteSampler))

	exporter, _ := stdouttrace.New()

	tp := trace.NewTracerProvider(
		trace.WithSampler(jaegerRemoteSampler),
		trace.WithSyncer(exporter), // for production usage, use trace.WithBatcher(exporter)
	)
	otel.SetTracerProvider(tp)

	fmt.Printf("\n* Initial Jaeger Remote Sampler: %v\n\n", time.Now())
	spewCfg := spew.ConfigState{
		Indent:                  "    ",
		DisablePointerAddresses: true,
		SortKeys:                true,
	}
	spewCfg.Dump(jaegerRemoteSampler)

	ticker := time.Tick(samplingRefreshInterval / 2)
	for {
		<-ticker
		fmt.Printf("\n* Jaeger Remote Sampler: %v\n\n", time.Now())
		spewCfg.Dump(jaegerRemoteSampler)
	}
}
