// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ot implements the ot-tracer-* propagator used by the default Tracer
// implementation from the OpenTracing project.
//
// Deprecated: OpenTracing is deprecated.
// v1.48.0 is the final planned release of this module.
// Migrate to native OpenTelemetry APIs and SDKs, and use
// [go.opentelemetry.io/otel/propagation.TraceContext] and
// [go.opentelemetry.io/otel/propagation.Baggage] for propagation.
// See https://opentelemetry.io/docs/compatibility/migration/opentracing/.
package ot
