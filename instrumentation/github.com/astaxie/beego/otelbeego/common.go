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

package otelbeego // import "go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego"

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/github.com/astaxie/beego/otelbeego/internal"
	"go.opentelemetry.io/otel/attribute"
)

// ------------------------------------------ Attribute Functions

// Template returns the template name as a KeyValue pair.
func Template(name string) attribute.KeyValue {
	return internal.TemplateKey.String(name)
}

// ------------------------------------------ OTel HTTP Types

// Filter returns true if the request should be traced.
type Filter func(*http.Request) bool

// SpanNameFormatter creates a custom span name from the operation and request object.
type SpanNameFormatter func(operation string, req *http.Request) string
