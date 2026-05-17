// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	v := Version()
	assert.NotEmpty(t, v)
	assert.Equal(t, v, SemVersion())
}
