// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOperationDuration(t *testing.T) {
	ctx := context.WithValue(t.Context(), spanTimestampKey{}, time.Now().Add(-time.Second))

	d, ok := OperationDuration(ctx)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, d, time.Second)
}

func TestOperationDurationWithoutStart(t *testing.T) {
	d, ok := OperationDuration(t.Context())
	assert.False(t, ok)
	assert.Zero(t, d)
}
