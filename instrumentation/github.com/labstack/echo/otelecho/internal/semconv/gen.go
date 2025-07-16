// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv // import "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho/internal/semconv"

// Generate semconv package:
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/bench_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=bench_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/common_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=common_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/env.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=env.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/env_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=env_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/httpconv.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=httpconv.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/httpconv_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=httpconv_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/httpconvtest_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=httpconvtest_test.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/util.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho\" }" --out=util.go
//go:generate gotmpl --body=../../../../../../../internal/shared/semconv/util_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/\" }" --out=util_test.go
