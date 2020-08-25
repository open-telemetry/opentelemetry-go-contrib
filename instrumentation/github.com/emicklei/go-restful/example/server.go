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
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful/v3"

	restfultrace "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	otelkv "go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer oteltrace.Tracer

type UserResource struct{}

func (u UserResource) WebService() *restful.WebService {
	ws := &restful.WebService{}

	ws.Path("/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{user-id}").To(u.getUser).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("integer").DefaultValue("1")).
		Writes(User{}). // on the response
		Returns(200, "OK", User{}).
		Returns(404, "Not Found", nil))
	return ws
}

func main() {
	initTracer()
	u := UserResource{}
	// create the Otel filter
	filter := restfultrace.OTelFilter("my-service")
	// use it
	restful.DefaultContainer.Filter(filter)
	restful.DefaultContainer.Add(u.WebService())

	_ = http.ListenAndServe(":8080", nil)
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
	tracer = otelglobal.TraceProvider().Tracer("go-restful-server", oteltrace.WithInstrumentationVersion("0.1"))
}

func (u UserResource) getUser(req *restful.Request, resp *restful.Response) {
	uid := req.PathParameter("user-id")
	_, span := tracer.Start(req.Request.Context(), "getUser", oteltrace.WithAttributes(otelkv.String("id", uid)))
	defer span.End()
	id, err := strconv.Atoi(uid)
	if err == nil && id >= 100 {
		_ = resp.WriteEntity(User{id})
		return
	}
	_ = resp.WriteErrorString(http.StatusNotFound, "User could not be found.")
}

type User struct {
	ID int `json:"id" description:"identifier of the user"`
}
