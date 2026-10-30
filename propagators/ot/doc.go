// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ot implements the ot-tracer-* propagator used by the default Tracer
// implementation from the OpenTracing project.
//
// Deprecated: OpenTracing is deprecated. This package will be removed in v1.49.0.
// Migrate to native OpenTelemetry APIs and SDKs, and use
// [go.opentelemetry.io/otel/propagation.TraceContext] and
// [go.opentelemetry.io/otel/propagation.Baggage] for propagation.
// See https://opentelemetry.io/docs/compatibility/migration/opentracing/.
package ot
