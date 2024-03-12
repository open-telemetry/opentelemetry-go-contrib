// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelgin instruments the github.com/gin-gonic/gin package.
//
// Currently there are two ways the code can be instrumented. One is
// instrumenting the routing of a received message (the Middleware function)
// and instrumenting the response generation through template evaluation (the
// HTML function).
package otelgin // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
