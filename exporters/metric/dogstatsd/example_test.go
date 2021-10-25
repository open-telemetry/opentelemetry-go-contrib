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

package dogstatsd_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func ExampleNew() {
	// Create a "server"
	wg := &sync.WaitGroup{}
	wg.Add(1)

	reader, writer := io.Pipe()

	go func() {
		var statLines []string

		defer wg.Done()
		defer func() { canonicalStats(statLines) }()

		for {
			var buf [4096]byte
			n, err := reader.Read(buf[:])
			if err == io.EOF {
				return
			} else if err != nil {
				log.Fatal("Read err: ", err)
			} else if n >= len(buf) {
				log.Fatal("Read small buffer: ", n)
			} else {
				statLines = append(statLines, string(buf[0:n]))
			}
		}
	}()

	// Create a meter
	cont, err := dogstatsd.NewExportPipeline(dogstatsd.Config{
		// The Writer field provides test support.
		Writer: writer,

		// In real code, use the URL field:
		//
		// URL: fmt.Sprint("unix://", path),
	}, controller.WithCollectPeriod(time.Minute), controller.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attribute.String("host", "name"))))
	if err != nil {
		log.Fatal("Could not initialize dogstatsd exporter:", err)
	}

	ctx := context.Background()

	key := attribute.Key("key")

	// cont implements the metric.MeterProvider interface:
	meter := cont.Meter("example")

	// Create and update a single counter:
	counter := metric.Must(meter).NewInt64Counter("a.counter")
	values := metric.Must(meter).NewInt64Histogram("a.values")

	values.Record(ctx, 50, key.String("value"))
	counter.Add(ctx, 100, key.String("value"))
	values.Record(ctx, 150, key.String("value"))

	// Flush the exporter, close the pipe, and wait for the reader.
	err = cont.Stop(context.Background())
	if err != nil {
		panic(err)
	}
	writer.Close()
	wg.Wait()

	// Output:
	// a.counter:100|c|#host:name,key:value
	// a.values:150|h|#host:name,key:value
	// a.values:50|h|#host:name,key:value
}

func canonicalStats(lines []string) {
	// split on newline and sort
	// concatenate, split on newline and sort
	var allStats string
	for _, s := range lines {
		allStats += s
	}
	// trim any trailing "\n"
	lines = strings.Split(strings.TrimSuffix(allStats, "\n"), "\n")
	sort.Strings(lines)
	for _, s := range lines {
		fmt.Println(s)
	}
}
