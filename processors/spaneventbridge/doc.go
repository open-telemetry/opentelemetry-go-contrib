// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package spaneventbridge provides a [go.opentelemetry.io/otel/sdk/log.Processor]
// that bridges log-based events back onto the current span as span events.
//
// A record is bridged when:
//
//   - the record has a non-empty event name,
//   - the current span is recording, and
//   - the record trace and span IDs match the current span.
//
// The bridged span event uses the log record event name and timestamp. Log
// attributes are copied onto the span event, and selected log record metadata
// is added using "log.record.*" attributes.
package spaneventbridge // import "go.opentelemetry.io/contrib/processors/spaneventbridge"
