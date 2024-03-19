// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	assert.IsType(t, &Handler{}, NewLogger().Handler())
}
