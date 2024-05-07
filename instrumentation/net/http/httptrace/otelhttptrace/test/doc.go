// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package test validates the otelhttptrace instrumentation with the default SDK.

This package is in a separate module from the instrumentation it tests to
isolate the dependency of the default SDK and not impose this as a transitive
dependency for users.
*/
package test // import "go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/test"
