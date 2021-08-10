package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/graphql/_example/graph/generated"
	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/graphql/_example/graph/model"
)

func (r *queryResolver) Ping(ctx context.Context) (*model.Pong, error) {
	return &model.Pong{ID: "Pong"}, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
