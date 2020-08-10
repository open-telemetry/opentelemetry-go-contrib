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
	"log"
	"os"

	"github.com/bradfitz/gomemcache/memcache"

	otelgomemcache "go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache"
	otelglobal "go.opentelemetry.io/otel/api/global"

	oteltracestdout "go.opentelemetry.io/otel/exporters/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	var host, port = os.Getenv("HOST"), "11211"

	initTracer()
	ctx := context.Background()

	c := otelgomemcache.NewClientWithTracing(
		memcache.New(
			host + ":" + port,
		),
	)

	t := otelglobal.Tracer("memcached-test")
	ctx, s := t.Start(ctx, "test-operations")
	doMemcacheOperations(ctx, c)
	s.End()
}

func doMemcacheOperations(ctx context.Context, c *otelgomemcache.Client) {
	err := c.WithContext(ctx).Add(&memcache.Item{
		Key:   "foo",
		Value: []byte("bar"),
	})
	if err != nil {
		log.Printf("Add failed: %s", err)
	}

	_, err = c.WithContext(ctx).Get("foo")
	if err != nil {
		log.Printf("Get failed: %s", err)
	}

	err = c.WithContext(ctx).Delete("baz")
	if err != nil {
		log.Printf("Delete failed: %s", err)
	}

	err = c.WithContext(ctx).DeleteAll()
	if err != nil {
		log.Printf("DeleteAll failed: %s", err)
	}
}

func initTracer() {
	exporter, err := oteltracestdout.NewExporter(oteltracestdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	cfg := sdktrace.Config{
		DefaultSampler: sdktrace.AlwaysSample(),
	}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(cfg),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		log.Fatal(err)
	}
	otelglobal.SetTraceProvider(tp)
}
