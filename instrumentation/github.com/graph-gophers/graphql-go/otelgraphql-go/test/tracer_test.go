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

package test

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const schemaString = `
	schema {
		query: Query
	}
	type Query {
		echo(message: String!): String!
		echo2(message: String!): String!
		echoError(): String!
	}
	`

type RootResolver struct{}

func (*RootResolver) Echo(args struct{ Message string }) string {
	return args.Message
}
func (r *RootResolver) Echo2(args struct{ Message string }) string {
	return r.Echo(args)
}
func (r *RootResolver) EchoError() (string, error) {
	return "", errors.New("echo error")
}

type fixture struct {
	SpanRecorder   *tracetest.SpanRecorder
	GraphQLHandler *relay.Handler
}

func newFixture() *fixture {
	sr := tracetest.NewSpanRecorder()
	tp := tracesdk.NewTracerProvider()
	tp.RegisterSpanProcessor(sr)

	tracer := otelgraphqlgo.NewOpenTelemetryTracer(otelgraphqlgo.WithTracerProvider(tp))

	opts := []graphql.SchemaOpt{
		graphql.Tracer(tracer),
		graphql.UseFieldResolvers(),
	}
	schema := graphql.MustParseSchema(schemaString, &RootResolver{}, opts...)

	handler := &relay.Handler{Schema: schema}

	return &fixture{
		SpanRecorder:   sr,
		GraphQLHandler: handler,
	}
}

func (f *fixture) getSpans(query string, vars string) []tracesdk.ReadOnlySpan {
	request := fmt.Sprintf(`{"query":"%s",
		"variables":%s}`, query, vars)
	body := strings.NewReader(request)
	r := httptest.NewRequest("GET", "/graphql", body)
	w := httptest.NewRecorder()

	f.GraphQLHandler.ServeHTTP(w, r)

	return f.SpanRecorder.Ended()
}

func TestForSingleFieldTrace(t *testing.T) {
	query := "query Echo($message: String!) { echo (message: $message) }"
	vars := "{\"message\": \"Hello\"}"
	spans := newFixture().getSpans(query, vars)

	require.Len(t, spans, 3)
	var hasValidationSpan, hasFieldSpan, hasRequestSpan bool
	var spanQuery, spanField string
	for _, span := range spans {
		details := getSpanDetails(span)
		if details.SpanType == ValidationSpan {
			hasValidationSpan = true
		}
		if details.SpanType == FieldSpan {
			hasFieldSpan = true
			spanField = details.Field
		}
		if details.SpanType == RequestSpan {
			hasRequestSpan = true
			spanQuery = details.Query
		}
	}
	assert.True(t, hasValidationSpan)
	assert.True(t, hasFieldSpan)
	assert.True(t, hasRequestSpan)
	assert.Equal(t, query, spanQuery)
	assert.Equal(t, "echo", spanField)
}

/*
func TestForTwoFieldTraces(t *testing.T) {
	query := "query Echo($message1: String!, $message2: String!) { echo (message: $message1)\\necho2 (message: $message2) }"
	vars := "{\"message1\": \"Hello\", \"message2\": \"World\"}}"
	spans := newFixture().getSpans(query, vars)

	require.Len(t, spans, 4)
	var hasValidationSpan, hasRequestSpan bool
	var fieldSpans int
	for _, span := range spans {
		details := getSpanDetails(span)
		if details.SpanType == ValidationSpan {
			hasValidationSpan = true
		}
		if details.SpanType == FieldSpan {
			fieldSpans++
		}
		if details.SpanType == RequestSpan {
			hasRequestSpan = true
		}
	}
	assert.True(t, hasValidationSpan)
	assert.Equal(t, fieldSpans, 2)
	assert.True(t, hasRequestSpan)
}
*/

func TestForValidationTraceWithError(t *testing.T) {
	query := "query { nonExistingFieldToTriggerValidationError }"
	vars := "{}"
	spans := newFixture().getSpans(query, vars)

	require.Len(t, spans, 1)
	details := getSpanDetails(spans[0])
	assert.Equal(t, details.SpanType, ValidationSpan)
	assert.True(t, details.HasError)
}

func TestForRequestTraceWithError(t *testing.T) {
	query := "query { echoError }"
	vars := "{}"
	spans := newFixture().getSpans(query, vars)

	require.Len(t, spans, 3)
	var hasFieldSpan, hasValidationSpan, hasRequestSpan, requestSpanHasError bool
	for _, span := range spans {
		details := getSpanDetails(span)
		if details.SpanType == ValidationSpan {
			hasValidationSpan = true
		}
		if details.SpanType == FieldSpan {
			hasFieldSpan = true
		}
		if details.SpanType == RequestSpan {
			hasRequestSpan = true
			requestSpanHasError = details.HasError
		}
	}
	assert.True(t, hasValidationSpan)
	assert.True(t, hasFieldSpan)
	assert.True(t, hasRequestSpan)
	assert.True(t, requestSpanHasError)
}

type spanType int

const (
	Unset          spanType = iota
	RequestSpan    spanType = iota
	ValidationSpan spanType = iota
	FieldSpan      spanType = iota
)

type spanDetails struct {
	SpanType spanType
	HasError bool
	Query    string
	Field    string
}

func getSpanDetails(span tracesdk.ReadOnlySpan) spanDetails {
	d := spanDetails{
		HasError: span.Status().Code.String() == "Error",
		SpanType: Unset,
	}
	for _, attr := range span.Attributes() {
		if attr.Key == "trace.operation" {
			switch attr.Value.AsString() {
			case "request":
				d.SpanType = RequestSpan
			case "validation":
				d.SpanType = ValidationSpan
			case "field":
				d.SpanType = FieldSpan
			}
		}
		if attr.Key == "graphql.query" {
			d.Query = attr.Value.AsString()
		}
		if attr.Key == "graphql.field" {
			d.Field = attr.Value.AsString()
		}
	}
	return d
}
