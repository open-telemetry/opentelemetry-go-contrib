// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test // import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin/internal/semconv/test"

// Generate semconv/test package:
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/common_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin\" }" --out=common_test.go
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/httpconv_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin\" }" --out=httpconv_test.go
//go:generate gotmpl --body=../../../../../../../../internal/shared/semconv/test/v1.20.0_test.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin\" }" --out=v1.20.0_test.go
