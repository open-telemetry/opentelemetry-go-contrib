// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package baggage implements the baggage span processor which duplicates
// onto a span the attributes found in Baggage in the parent context at
// the moment the span is started.
package baggage // import "go.opentelemetry.io/contrib/processors/baggage"
