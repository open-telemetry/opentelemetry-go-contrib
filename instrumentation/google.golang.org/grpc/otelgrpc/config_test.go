// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	grpc_codes "google.golang.org/grpc/codes"
)

func TestWithNonErrorCodes(t *testing.T) {
	t.Run("configures codes and wires it correctly", func(t *testing.T) {
		c := newConfig([]Option{WithNonErrorCodes(grpc_codes.NotFound)})

		if c.NonErrorCodes == nil {
			t.Fatal("expected NonErrorCodes to be configured, got nil")
		}
		assert.Len(t, c.NonErrorCodes, 1)
		assert.Contains(t, c.NonErrorCodes, grpc_codes.NotFound)
	})

	t.Run("no input is ignored", func(t *testing.T) {
		c := newConfig([]Option{WithNonErrorCodes()})
		assert.Empty(t, c.NonErrorCodes)
	})

	t.Run("repeated options are additive", func(t *testing.T) {
		c := newConfig([]Option{
			WithNonErrorCodes(grpc_codes.NotFound),
			WithNonErrorCodes(grpc_codes.AlreadyExists),
			WithNonErrorCodes(),
		})
		assert.Equal(t, newConfig([]Option{
			WithNonErrorCodes(grpc_codes.NotFound, grpc_codes.AlreadyExists),
		}).NonErrorCodes, c.NonErrorCodes)
		assert.Len(t, c.NonErrorCodes, 2)
	})

	t.Run("duplicate entries are ignored", func(t *testing.T) {
		c := newConfig([]Option{WithNonErrorCodes(grpc_codes.NotFound, grpc_codes.NotFound)})
		assert.Len(t, c.NonErrorCodes, 1)
		assert.Contains(t, c.NonErrorCodes, grpc_codes.NotFound)
	})

	t.Run("mutation of input-slice does not affect config", func(t *testing.T) {
		slice := []grpc_codes.Code{grpc_codes.NotFound}
		c := newConfig([]Option{WithNonErrorCodes(slice...)})
		assert.Len(t, c.NonErrorCodes, 1)
		assert.Contains(t, c.NonErrorCodes, grpc_codes.NotFound)

		slice[0] = grpc_codes.Internal // Attempt to mutate the input slice after the config has been created
		assert.Len(t, c.NonErrorCodes, 1)
		assert.Contains(t, c.NonErrorCodes, grpc_codes.NotFound)
		assert.NotContains(t, c.NonErrorCodes, grpc_codes.Internal)
	})
}
