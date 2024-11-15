// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otellogrus // import "go.opentelemetry.io/contrib/bridges/otellogrus"

// Generate convert:
//go:generate gotmpl --body=../../internal/shared/logutil/convert_test.go.tmpl "--data={ \"pkg\": \"otellogrus\" }" --out=convert_test.go
//go:generate gotmpl --body=../../internal/shared/logutil/convert.go.tmpl "--data={ \"pkg\": \"otellogrus\" }" --out=convert.go
