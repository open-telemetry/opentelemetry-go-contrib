// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package test provides semantic convention tests for otelhttp.
package test // import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/internal/semconv/test"

// Generate semconv/test package:
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/test/common_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp\" }" --out=common_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/test/httpconv_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp\" }" --out=httpconv_test.go
