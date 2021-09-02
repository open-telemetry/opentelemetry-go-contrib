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
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"go.opentelemetry.io/contrib/instrumentation/github.com/graph-gophers/graphql-go/otelgraphqlgo"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const schemaString = `
	schema {
		query: Query
		mutation: Mutation
	}
	type User {
		userID: ID!
		fullName: String!
		username: String!
		organization: String!
	}
	input UserInput {
		fullName: String!
		username: String!
		organization: String!
	}
	type Query {
		user(username: String!): User
		users: [User!]!
		usersOfOrganization(organization: String!): [User!]!
	}
	type Mutation {
		createUser(userInput: UserInput!): User
	}
`

type RootResolver struct{}

type User struct {
	UserID       graphql.ID
	FullName     string
	Username     string
	Organization string
}

type UserInput struct {
	FullName     string
	Username     string
	Organization string
}

var users = []User{
	{graphql.ID("1"), "John Smith", "johnsmith", "HR"},
	{graphql.ID("2"), "Jone Doe", "jonedoe", "IT"},
	{graphql.ID("3"), "Jane Doe", "janedoe", "Marketing"},
}

func (*RootResolver) User(args struct{ Username string }) (*User, error) {
	for _, u := range users {
		if u.Username == args.Username {
			return &u, nil
		}
	}
	return nil, nil
}

func (*RootResolver) Users() ([]User, error) {
	return users, nil
}

func (*RootResolver) UsersOfOrganization(args struct{ Organization string }) ([]User, error) {
	return []User{}, errors.New("intentional error")
}

func (*RootResolver) CreateUser(args struct{ UserInput UserInput }) (*User, error) {
	user := User{
		UserID:       graphql.ID(uuid.NewString()),
		FullName:     args.UserInput.FullName,
		Username:     args.UserInput.Username,
		Organization: args.UserInput.Organization,
	}
	users = append(users, user)
	return &user, nil
}

var schema *graphql.Schema

func main() {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("graphql-server"),
		)),
	)
	tracer := otelgraphqlgo.NewOpenTelemetryTracer(otelgraphqlgo.WithTracerProvider(tp))

	defer func() {
		if err = tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	opts := []graphql.SchemaOpt{
		graphql.Tracer(tracer),
		graphql.UseFieldResolvers(),
	}
	schema = graphql.MustParseSchema(schemaString, &RootResolver{}, opts...)

	http.Handle("/graphql", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
