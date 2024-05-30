// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelmongo instruments go.mongodb.org/mongo-driver/mongo.
//
// This package is compatible with v0.2.0 of
// go.mongodb.org/mongo-driver/mongo.
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace
// requests.
//
// This code was originally based on the following:
// - https://github.com/DataDog/dd-trace-go/tree/02f0449efa3cb382d499fadc873957385dcb2192/contrib/go.mongodb.org/mongo-driver/mongo
// - https://github.com/DataDog/dd-trace-go/tree/v1.23.3/ddtrace/ext
//
// Deprecated: otelmongo has no Code Owner.
// After August 21, 2024, it may no longer be supported and may stop
// receiving new releases unless a new Code Owner is found. See
// [this issue] if you would like to become the Code Owner of this module.
//
// [this issue]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5551
package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
