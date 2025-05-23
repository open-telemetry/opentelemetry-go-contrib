// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test"

// Version is the current release version of the gRPC instrumentation test module.
func Version() string {
	return "0.61.0"
	// This string is updated by the pre_release.sh script during release
}
