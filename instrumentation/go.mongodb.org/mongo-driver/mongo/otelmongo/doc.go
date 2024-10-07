// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelmongo instruments go.mongodb.org/mongo-driver/mongo.
//
// This package is compatible with v0.2.0 of
// go.mongodb.org/mongo-driver/mongo.
//
// NewMonitor will return an event.CommandMonitor which is used to trace
// requests.
//
// This code was originally based on the following:
//   - https://github.com/DataDog/dd-trace-go/tree/02f0449efa3cb382d499fadc873957385dcb2192/contrib/go.mongodb.org/mongo-driver/mongo
//   - https://github.com/DataDog/dd-trace-go/tree/v1.23.3/ddtrace/ext
//
// The "OTEL_SEMCONV_STABILITY_OPT_IN" environment variable  can be used to opt
// into semconv/v1.26.0:
//   - "mongo/v1.26.0": emit v1.26.0 semantic conventions
//   - "mongo/v1.17.0": emit v1.17.0 (default) semantic conventions
//   - "mongo/dup": emit the stable version (v1.17.0) and all other supported semantic conventions
//
// "mongo/dup" takes precedence over "mongo/v*". By default, otelmongo only emits v1.17.0.
//
// For example, the following will use v1.26.0 for otelmongo and duplicate
// attributes (old + new) for otelhttp:
//
//	export OTEL_SEMCONV_STABILITY_OPT_IN="mongo/v1.26.0,http/dup"
package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
