// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package mongo_test

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongotrace "go.opentelemetry.io/contrib/go.mongodb.org/mongo-driver/mongo"
)

func Example() {
	// connect to MongoDB
	opts := options.Client()
	opts.Monitor = mongotrace.NewMonitor()
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
