// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build go1.24

package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import "encoding"

var (
	_ encoding.TextAppender = Severity(0)         // Ensure Severity implements encoding.TextAppender.
	_ encoding.TextAppender = (*SeverityVar)(nil) // Ensure Severity implements encoding.TextAppender.
)
