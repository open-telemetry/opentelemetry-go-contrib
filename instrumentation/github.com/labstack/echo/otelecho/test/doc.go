// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package test validates the otelecho instrumentation with the default SDK.

This package is in a separate module from the instrumentation it tests to
isolate the dependency of the default SDK and not impose this as a transitive
dependency for users.

Deprecated: otelecho has no Code Owner.
After August 21, 2024, it may no longer be supported and may stop
receiving new releases unless a new Code Owner is found. See
[this issue] if you would like to become the Code Owner of this module.

[this issue]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5550
*/
package test // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho/test"
