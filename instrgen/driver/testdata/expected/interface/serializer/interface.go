// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package serializer

import __atel_context "context"

type Serializer interface {
	Serialize(__atel_tracing_ctx __atel_context.Context)
}
