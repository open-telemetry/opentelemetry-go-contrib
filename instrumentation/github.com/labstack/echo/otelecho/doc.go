// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelecho instruments the labstack/echo package
// (https://github.com/labstack/echo).
//
// Currently only the routing of a received message can be instrumented. To do
// so, use the Middleware function.
//
// Deprecated: Use [github.com/labstack/echo-opentelemetry] instead. See
// [MIGRATION.md] for known incompatibilities.
//
// [github.com/labstack/echo-opentelemetry]: https://github.com/labstack/echo-opentelemetry
// [MIGRATION.md]: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/labstack/echo/otelecho/MIGRATION.md
package otelecho // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
