// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
