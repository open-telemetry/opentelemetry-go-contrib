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
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful/v3"

	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer oteltrace.Tracer

type userResource struct{}

func (u userResource) WebService() *restful.WebService {
	ws := &restful.WebService{}

	ws.Path("/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{user-id}").To(u.getUser).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("integer").DefaultValue("1")).
		Writes(user{}). // on the response
		Returns(200, "OK", user{}).
		Returns(404, "Not Found", nil))
	return ws
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	u := userResource{}
	// create the Otel filter
	filter := otelrestful.OTelFilter("my-service")
	// use it
	restful.DefaultContainer.Filter(filter)
	restful.DefaultContainer.Add(u.WebService())

	_ = http.ListenAndServe(":8080", nil)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}))
	tracer = otel.GetTracerProvider().Tracer("go-restful-server", oteltrace.WithInstrumentationVersion("0.1"))
	return tp, nil
}

func (u userResource) getUser(req *restful.Request, resp *restful.Response) {
	uid := req.PathParameter("user-id")
	_, span := tracer.Start(req.Request.Context(), "getUser", oteltrace.WithAttributes(attribute.String("id", uid)))
	defer span.End()
	id, err := strconv.Atoi(uid)
	if err == nil && id >= 100 {
		_ = resp.WriteEntity(user{id})
		return
	}
	_ = resp.WriteErrorString(http.StatusNotFound, "User could not be found.")
}

type user struct {
	ID int `json:"id" description:"identifier of the user"`
}
