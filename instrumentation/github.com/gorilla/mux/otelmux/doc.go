// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelmux instruments the github.com/gorilla/mux package.
//
// Currently only the routing of a received message can be instrumented. To do
// it, use the Middleware function.
//
// Deprecated: otelmux has no Code Owner.
// After August 21, 2024, it may no longer be supported and may stop
// receiving new releases unless a new Code Owner is found. See
// [this issue] if you would like to become the Code Owner of this module.
//
// [this issue]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5549
package otelmux // import "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
