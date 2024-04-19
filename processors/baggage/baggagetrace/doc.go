// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package baggagetrace implements the baggage span processor which duplicates
// onto a span the attributes found in Baggage in the parent context at
// the moment the span is started.
package baggagetrace // import "go.opentelemetry.io/contrib/processors/baggage/baggagetrace"
