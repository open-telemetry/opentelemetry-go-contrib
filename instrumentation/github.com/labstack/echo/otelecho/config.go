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

package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// config is used to configure the mux middleware.
type config struct {
	otelhttpOptions         []otelhttp.Option
	noRouteTagFromPath      bool
	noPathSpanNameFormatter bool
	hasSpanNameFormatter    bool
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(opts ...Option) *config {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}

	// If user hasn't passed a span name formatter and the default hasn't been disabled, set the default as PathSpanNameFormatter
	if !c.hasSpanNameFormatter && !c.noPathSpanNameFormatter {
		c.otelhttpOptions = append(c.otelhttpOptions, otelhttp.WithSpanNameFormatter(PathSpanNameFormatter))
	}

	return c
}

// WithSkipper specifies a skipper for allowing requests to skip generating spans.
func WithSkipper(skipper func(c echo.Context) bool) Option {
	return optionFunc(func(conf *config) {
		conf.otelhttpOptions = append(conf.otelhttpOptions, otelhttp.WithFilter(func(request *http.Request) bool {
			c := request.Context().Value(echoContextCtxKey).(echo.Context)
			return skipper(c)
		}))
	})
}

// WithoutRouteTagFromPath removes a middleware from the chain to tag all routes with echo.Context.Path().
func WithoutRouteTagFromPath() Option {
	return optionFunc(func(c *config) {
		c.noRouteTagFromPath = true
	})
}

func WithoutPathSpanNameFormatter() Option {
	return optionFunc(func(c *config) {
		c.noPathSpanNameFormatter = true
	})
}
