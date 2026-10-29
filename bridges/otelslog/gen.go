// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog

// Generate convert:
//go:generate gotmpl --body=../../internal/shared/logutil/convert_test.go.tmpl "--data={ \"pkg\": \"otelslog\" }" --out=convert_test.go
//go:generate gotmpl --body=../../internal/shared/logutil/convert.go.tmpl "--data={ \"pkg\": \"otelslog\" }" --out=convert.go
