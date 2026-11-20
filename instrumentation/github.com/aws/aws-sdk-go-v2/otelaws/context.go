// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"time"
)

// OperationDuration returns the time elapsed since the AWS SDK operation of
// ctx started, which is the start of its span.
//
// It returns false if ctx is not the context passed by
// the middlewares appended by AppendMiddlewares.
func OperationDuration(ctx context.Context) (time.Duration, bool) {
	start, ok := ctx.Value(spanTimestampKey{}).(time.Time)
	if !ok {
		return 0, false
	}
	return time.Since(start), true
}
