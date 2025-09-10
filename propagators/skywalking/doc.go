// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package skywalking implements the SkyWalking propagator specification.
//
// SkyWalking uses sw8 headers for cross-process propagation of trace context,
// sw8-correlation headers for propagating correlation data, and sw8-x extension
// headers for tracing mode control and transmission latency calculation.
// The propagator extracts and injects trace context using the SkyWalking v3 format
// and automatically handles correlation data through OpenTelemetry baggage.
//
// For more information about SkyWalking propagation, see:
//   - SW8 Headers: https://skywalking.apache.org/docs/main/latest/en/api/x-process-propagation-headers-v3/
//   - SW8-Correlation Headers: https://skywalking.apache.org/docs/main/latest/en/api/x-process-correlation-headers-v1/
package skywalking // import "go.opentelemetry.io/contrib/propagators/skywalking"
