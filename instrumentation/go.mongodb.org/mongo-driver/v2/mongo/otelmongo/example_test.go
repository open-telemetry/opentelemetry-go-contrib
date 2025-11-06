// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo_test

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

func Example() {
	// connect to MongoDB
	opts := options.Client()
	opts.Monitor = otelmongo.NewMonitor()
	opts.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	db := client.Database("example")
	inventory := db.Collection("inventory")

	_, err = inventory.InsertOne(context.TODO(), bson.D{
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

func ExampleWithSpanNameFormatter() {
	// connect to MongoDB
	opts := options.Client()
	opts.Monitor = otelmongo.NewMonitor(
		otelmongo.WithSpanNameFormatter(func(event *event.CommandStartedEvent) string {
			// optionally, the collection name can be extracted for more
			// descriptive span names; see the extractCollection helper function
			// in the otelmongo package for an example of how to do this.
			return fmt.Sprintf("my-prefix-%s", event.CommandName)
		}),
	)
	opts.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	db := client.Database("mystore")
	inventory := db.Collection("inventory")

	_, err = inventory.InsertOne(context.TODO(), bson.D{
		{Key: "item", Value: "canvas"},
		// [..]
	})
	if err != nil {
		panic(err)
	}
}
