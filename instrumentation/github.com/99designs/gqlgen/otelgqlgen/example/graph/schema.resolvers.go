package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/example/graph/generated"
	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/example/graph/model"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func (r *queryResolver) GetUser(ctx context.Context, id string) (*model.User, error) {
	_, span := r.Tracer.Start(ctx, "getUser", oteltrace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	name := "unknown"
	if id == "123" {
		name = "otelgqlgen tester"
	}

	return &model.User{
		ID:   id,
		Name: name,
	}, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
