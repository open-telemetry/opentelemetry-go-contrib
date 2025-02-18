// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test // import "go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace/internal/semconv/test"

// Generate semconv/test package:
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/common_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace\" }" --out=common_test.go
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/httpconv_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace\" }" --out=httpconv_test.go
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/v1.20.0_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace\" }" --out=v1.20.0_test.go
