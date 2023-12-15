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

package otelmongo_test

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

func Example() {
	// connect to MongoDB
	opts := options.Client()
	opts.Monitor = otelmongo.NewMonitor()
	opts.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
	db := client.Database("example")
	inventory := db.Collection("inventory")

	_, err = inventory.InsertOne(context.Background(), bson.D{
		{Key: "item", Value: "canvas"},
		{Key: "qty", Value: 100},
		{Key: "attributes", Value: bson.A{"cotton"}},
		{Key: "size", Value: bson.D{
			{Key: "h", Value: 28},
			{Key: "w", Value: 35.5},
			{Key: "uom", Value: "cm"},
		}},
	})
	if err != nil {
		panic(err)
	}
}

func ExampleLimitedStatementMarshaller() {
	// connect to MongoDB
	opts := options.Client()
	opts.Monitor = otelmongo.NewMonitor(
		otelmongo.WithMarshaller(otelmongo.NewLimitedStatementMarshaller(50)),
	)
	opts.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
	db := client.Database("example")
	inventory := db.Collection("inventory")

	_, err = inventory.InsertOne(context.Background(), bson.D{
		{Key: "item", Value: "canvas"},
		{Key: "qty", Value: 100},
		{Key: "attributes", Value: bson.A{"cotton"}},
		{Key: "size", Value: bson.D{
			{Key: "h", Value: 28},
			{Key: "w", Value: 35.5},
			{Key: "uom", Value: "cm"},
		}},
	})
	if err != nil {
		panic(err)
	}
}
