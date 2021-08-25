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

package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/_example/graph/generated"
	"go.opentelemetry.io/opentelemetry-go-contrib/instrumentation/github.com/99designs/gqlgen/otelgqlgen/_example/graph/model"
)

// Ping processing when the client calls GQL query ping
func (r *queryResolver) Ping(ctx context.Context) (*model.Pong, error) {
	return &model.Pong{ID: "Pong"}, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
