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

package beego

import (
	"net/http"

	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http"

	"github.com/astaxie/beego"
)

// defaultSpanNameFormatter is the default formatter for spans created with
// the beego integration. Returns the request URL path.
func defaultSpanNameFormatter(operation string, req *http.Request) string {
	return req.URL.Path
}

// NewOTelBeegoMiddleWare creates a MiddleWare that provides OpenTelemetry
// tracing and metrics to a Beego web app. 
// Name is the http handler operation name.
// The OTelBeegoMiddleWare can be configured using the provided Options.
func NewOTelBeegoMiddleWare(name string, options ...Option) beego.MiddleWare {
	cfg := configure(options...)

	httpOptions := []otelhttp.Option{
		otelhttp.WithTracer(cfg.tracer),
		otelhttp.WithMeter(cfg.meter),
		otelhttp.WithPropagators(cfg.propagators),
	}

	for _, f := range cfg.filters {
		httpOptions = append(
			httpOptions,
			otelhttp.WithFilter(otelhttp.Filter(f)),
		)
	}

	if cfg.formatter != nil {
		httpOptions = append(httpOptions, otelhttp.WithSpanNameFormatter(cfg.formatter))
	}

	return func(handler http.Handler) http.Handler {
		return otelhttp.NewHandler(
			handler,
			name,
			httpOptions...,
		)
	}
}
