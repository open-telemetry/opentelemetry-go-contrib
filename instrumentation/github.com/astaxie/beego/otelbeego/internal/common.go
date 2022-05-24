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

package internal // import "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/internal"

import (
	"go.opentelemetry.io/otel/attribute"
)

// ContextKey is a key for a value in a context.Context,
// used as it is not recommended to use basic types as keys.
type ContextKey string

const (
	// CtxRouteTemplateKey is the context key used for a route template.
	CtxRouteTemplateKey = ContextKey("x-opentelemetry-route-template")

	// RenderTemplateSpanName is the span name for the beego.Controller.Render
	// operation.
	RenderTemplateSpanName = "beego.render.template"
	// RenderStringSpanName is the span name for the
	// beego.Controller.RenderString operation.
	RenderStringSpanName = "beego.render.string"
	// RenderStringSpanName is the span name for the
	// beego.Controller.RenderBytes operation.
	RenderBytesSpanName = "beego.render.bytes"

	// TemplateKey is used to describe the beego template used.
	TemplateKey = attribute.Key("go.template")
)
