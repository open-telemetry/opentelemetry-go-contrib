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

	"go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache"
	oteltrace "go.opentelemetry.io/otel/trace"

	oteltracestdout "go.opentelemetry.io/otel/exporters/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	var host, port = os.Getenv("HOST"), "11211"

	tp := initTracer()
	ctx := context.Background()

	c := otelmemcache.NewClientWithTracing(
		memcache.New(
			host+":"+port,
		),
		otelmemcache.WithTracerProvider(tp),
	)

	ctx, s := tp.Tracer("example-tracer").Start(ctx, "test-operations")
	doMemcacheOperations(ctx, c)
	s.End()
}

func doMemcacheOperations(ctx context.Context, c *otelmemcache.Client) {
	cc := c.WithContext(ctx)

	err := cc.Add(&memcache.Item{
		Key:   "foo",
		Value: []byte("bar"),
	})
	if err != nil {
		log.Printf("Add failed: %s", err)
	}

	_, err = cc.Get("foo")
	if err != nil {
		log.Printf("Get failed: %s", err)
	}

	err = cc.Delete("baz")
	if err != nil {
		log.Printf("Delete failed: %s", err)
	}

	err = cc.DeleteAll()
	if err != nil {
		log.Printf("DeleteAll failed: %s", err)
	}
}

func initTracer() oteltrace.TracerProvider {
	exporter, err := oteltracestdout.NewExporter(oteltracestdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		log.Fatal(err)
	}

	return tp
}
