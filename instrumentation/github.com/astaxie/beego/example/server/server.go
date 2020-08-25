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
	"log"

	"github.com/astaxie/beego"

	otelbeego "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

type ExampleController struct {
	beego.Controller
}

func (c *ExampleController) Get() {
	ctx := c.Ctx.Request.Context()
	span := trace.SpanFromContext(ctx)
	span.AddEvent(ctx, "handling this...")
	c.Ctx.WriteString("Hello, world!")
}

func (c *ExampleController) Template() {
	c.TplName = "hello.tpl"
	// Render the template file with tracing enabled
	if err := otelbeego.Render(&c.Controller); err != nil {
		c.Abort("500")
	}
}

func initTracer() {
	// Create stdout exporter to be able to retrieve
	// the collected spans.
	exporter, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(resource.New(semconv.ServiceNameKey.String("ExampleService"))))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

func main() {
	initTracer()

	// To enable tracing on template rendering, disable autorender
	beego.BConfig.WebConfig.AutoRender = false

	beego.Router("/hello", &ExampleController{})
	beego.Router("/", &ExampleController{}, "get:Template")

	mware := otelbeego.NewOTelBeegoMiddleWare("beego-example")

	beego.RunWithMiddleWares(":7777", mware)

}
