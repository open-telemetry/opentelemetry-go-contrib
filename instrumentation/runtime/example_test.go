// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime_test

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
)

func Example() {
	// This reader is used as a stand-in for a reader that will actually export
	// data. See https://pkg.go.dev/go.opentelemetry.io/otel/exporters for
	// exporters that can be used as or with readers.
	reader := metric.NewManualReader(
		// Add the runtime producer to get histograms from the Go runtime.
		metric.WithProducer(runtime.NewProducer()),
	)
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetMeterProvider(provider)

	// Start go runtime metric collection.
	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		log.Fatal(err)
	}
}
