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
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/oteltest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otel.SetTracerProvider(oteltest.NewTracerProvider())

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*oteltest.Tracer)
		require.True(t, ok)
		assert.Equal(t, tracerName, mockTracer.Name)
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(Middleware("foobar"))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
}

func TestChildSpanFromCustomTracer(t *testing.T) {
	provider := oteltest.NewTracerProvider()

	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := oteltrace.SpanFromContext(ctx)
		_, ok := span.(*oteltest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*oteltest.Tracer)
		require.True(t, ok)
		assert.Equal(t, tracerName, mockTracer.Name)
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})
	srv.Use(Middleware("foobar", WithTracerProvider(provider)))

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
}

func TestGetSpanNotInstrumented(t *testing.T) {
	srv := newMockServer(func(ctx context.Context) (interface{}, error) {
		span := oteltrace.SpanFromContext(ctx)
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
	})

	r := httptest.NewRequest("GET", "/foo?query={name}", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, r)
}

// newMockServer provides a server for use in resolver error tests that isn't relying on generated code.
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
	})
	srv.AddTransport(&transport.GET{})

	return srv
}
