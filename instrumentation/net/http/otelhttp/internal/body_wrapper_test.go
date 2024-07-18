// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyWrapper(t *testing.T) {
	bw := NewBodyWrapper(io.NopCloser(strings.NewReader("hello world")), func(int64) {})

	data, err := io.ReadAll(bw)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))

	assert.Equal(t, int64(11), bw.BytesRead())
	assert.Equal(t, io.EOF, bw.Error())
}
