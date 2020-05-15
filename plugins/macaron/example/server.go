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

	"gopkg.in/macaron.v1"

	macarontrace "go.opentelemetry.io/contrib/plugins/macaron"

	otelglobal "go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	oteltracestdout "go.opentelemetry.io/otel/exporters/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = otelglobal.Tracer("macaron-server")

func main() {
	initTracer()
	m := macaron.Classic()
	m.Use(macarontrace.Middleware("my-server"))
	m.Get("/users/:id", func(ctx *macaron.Context) string {
		id := ctx.Params("id")
		name := getUser(ctx.Req.Context(), id)
		return name
	})
	m.Run()
}

func initTracer() {
	exporter, err := oteltracestdout.NewExporter(oteltracestdout.Options{PrettyPrint: true})
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

func getUser(ctx context.Context, id string) string {
	_, span := tracer.Start(ctx, "getUser", oteltrace.WithAttributes(kv.String("id", id)))
	defer span.End()
	if id == "123" {
		return "macarontrace tester"
	}
	return "unknown"
}
