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

package otelsql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib"
)

func TestNewConfig(t *testing.T) {
	cfg := newConfig("db", WithSpanOptions(SpanOptions{Ping: true}))
	assert.Equal(t, config{
		TracerProvider: otel.GetTracerProvider(),
		Tracer: otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(contrib.SemVersion()),
		),
		SpanOptions: SpanOptions{Ping: true},
		DBSystem:    "db",
		Attributes: []attribute.KeyValue{
			semconv.DBSystemKey.String(cfg.DBSystem),
		},
		SpanNameFormatter: &defaultSpanNameFormatter{},
	}, cfg)
}
