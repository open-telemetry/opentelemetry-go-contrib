// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelmongo instruments go.mongodb.org/mongo-driver/v2/mongo.
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace
// requests.
//
// This code was originally based on the following:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/tree/323e373a6c15ae310bdd0617e3ed52d8cb8e4e6f/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo
package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
