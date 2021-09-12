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

package otelgqlgen

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	testQueryName     = "NamedQuery"
	namelessQueryName = "nameless-operation"
	testComplexity    = 5
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar"))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, namelessQueryName, codes.Unset)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestChildSpanFromGlobalTracerWithNamed(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar"))

	body := strings.NewReader(fmt.Sprintf("{\"operationName\":\"%s\",\"variables\":{},\"query\":\"query %s {\\n  name\\n}\\n\"}", testQueryName, testQueryName))
	r := httptest.NewRequest("POST", "/foo", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, testQueryName, codes.Unset)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar", otelgqlgen.WithTracerProvider(provider)))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, namelessQueryName, codes.Unset)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestChildSpanWithComplexityExtension(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar", otelgqlgen.WithComplexityExtensionName("APQ")))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, namelessQueryName, codes.Unset)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestGetSpanNotInstrumented(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if span.IsValid() {
			t.Fatalf("unexpected span: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestChildSpanFromGlobalTracerWithError(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServerError(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar"))
	var gqlErrors gqlerror.List
	var respErrors gqlerror.List
	srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		resp := next(ctx)
		gqlErrors = graphql.GetErrors(ctx)
		respErrors = resp.Errors
		return resp
	})

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, namelessQueryName, codes.Error)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, len(gqlErrors))
	assert.Equal(t, gqlErrors, respErrors)
}

func TestChildSpanFromGlobalTracerWithComplexity(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(otelgqlgen.Middleware("foobar"))
	srv.Use(extension.FixedComplexityLimit(testComplexity))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)

	testSpans(t, spanRecorder, namelessQueryName, codes.Unset)
	// second span because it's response span where stored RequestComplexityLimit attribute
	attributes := spanRecorder.Ended()[1].Attributes()
	var found bool
	for _, a := range attributes {
		if a.Key == ("gql.request.complexityLimit") {
			found = true
			assert.Equal(t, int(a.Value.AsInt64()), testComplexity)
		}
	}

	assert.True(t, found)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// newMockServer provides a server for use in resolver tests that isn't relying on generated code.
// It isn't a perfect reproduction of a generated server, but it aims to be good enough to
// test the handler package without relying on codegen.
func newMockServer(resolver func(ctx context.Context) (interface{}, error)) *handler.Server {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			name: String!
			find(id: Int!): String!
		}
		type Mutation {
			name: String!
		}
		type Subscription {
			name: String!
		}
	`})
	srv := handler.New(&graphql.ExecutableSchemaMock{
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			rc := graphql.GetOperationContext(ctx)
			switch rc.Operation.Operation {
			case ast.Query:
				ran := false
				return func(ctx context.Context) *graphql.Response {
					if ran {
						return nil
					}
					ran = true
					// Field execution happens inside the generated code, lets simulate some of it.
					ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
						Object: "Query",
						Field: graphql.CollectedField{
							Field: &ast.Field{
								Name:       "name",
								Alias:      "alias",
								Definition: schema.Types["Query"].Fields.ForName("name"),
								ObjectDefinition: &ast.Definition{
									Kind:        "kind",
									Description: "description",
									Name:        "name",
								},
							},
						},
					})
					res, err := graphql.GetOperationContext(ctx).ResolverMiddleware(ctx, resolver)
					if err != nil {
						panic(err)
					}
					return res.(*graphql.Response)
				}
			default:
				return graphql.OneShot(graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
		ComplexityFunc: func(typeName string, fieldName string, childComplexity int, args map[string]interface{}) (int, bool) {
			return childComplexity, true
		},
	})
	srv.AddTransport(&transport.GET{})
	srv.AddTransport(&transport.POST{})

	return srv
}

// newMockServerError provides a server for use in resolver error tests that isn't relying on generated code.
// It isnt a perfect reproduction of a generated server, but it aims to be good enough to
// test the handler package without relying on codegen.
func newMockServerError(resolver func(ctx context.Context) (interface{}, error)) *handler.Server {
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		type Query {
			name: String!
		}
	`})
	srv := handler.New(&graphql.ExecutableSchemaMock{
		ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
			rc := graphql.GetOperationContext(ctx)
			switch rc.Operation.Operation {
			case ast.Query:
				ran := false
				return func(ctx context.Context) *graphql.Response {
					if ran {
						return nil
					}
					ran = true
					// Field execution happens inside the generated code, lets simulate some of it.
					ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
						Object: "Query",
						Field: graphql.CollectedField{
							Field: &ast.Field{
								Name:       "name",
								Alias:      "alias",
								Definition: schema.Types["Query"].Fields.ForName("name"),
								ObjectDefinition: &ast.Definition{
									Kind:        "kind",
									Description: "description",
									Name:        "name",
								},
							},
						},
					})
					graphql.AddError(ctx, fmt.Errorf("resolver error"))

					res, err := graphql.GetOperationContext(ctx).ResolverMiddleware(ctx, resolver)
					if err != nil {
						panic(err)
					}
					return res.(*graphql.Response)
				}
			default:
				return graphql.OneShot(graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	})
	srv.AddTransport(&transport.GET{})

	return srv
}

func testSpans(t *testing.T, spanRecorder *tracetest.SpanRecorder, spanName string, spanCode codes.Code) {
	spans := spanRecorder.Ended()
	if got, expected := len(spans), 2; got != expected {
		t.Fatalf("got %d spans, expected %d", got, expected)
	}
	responseSpan := spans[1]
	if !responseSpan.SpanContext().IsValid() {
		t.Fatalf("invalid span created: %#v", responseSpan.SpanContext())
	}

	if responseSpan.Name() != spanName {
		t.Errorf("expected name on span %s; got: %q", spanName, responseSpan.Name())
	}

	for _, s := range spanRecorder.Ended() {
		assert.Equal(t, s.Status().Code, spanCode)
	}
}
