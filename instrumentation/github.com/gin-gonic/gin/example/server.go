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
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	gintrace "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = otelglobal.Tracer("gin-server")

func main() {
	initTracer()
	r := gin.New()
	r.Use(gintrace.Middleware("my-server"))
	tmplName := "user"
	tmplStr := "user {{ .name }} (id {{ .id }})\n"
	tmpl := template.Must(template.New(tmplName).Parse(tmplStr))
	r.SetHTMLTemplate(tmpl)
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		name := getUser(c.Request.Context(), id)
		gintrace.HTML(c, http.StatusOK, tmplName, gin.H{
			"name": name,
			"id":   id,
		})
	})
	_ = r.Run(":8080")
}

func initTracer() {
	exporter, err := stdout.NewExporter(stdout.WithPrettyPrint())
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
	_, span := tracer.Start(ctx, "getUser", oteltrace.WithAttributes(label.String("id", id)))
	defer span.End()
	if id == "123" {
		return "gintrace tester"
	}
	return "unknown"
}
