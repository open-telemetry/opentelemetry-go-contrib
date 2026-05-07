// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestLabelerFromContext(t *testing.T) {
	t.Run("no labeler in context", func(t *testing.T) {
		l, ok := LabelerFromContext(context.Background())
		assert.False(t, ok)
		require.NotNil(t, l)
		assert.Empty(t, l.Get())
	})

	t.Run("labeler set via ContextWithLabeler", func(t *testing.T) {
		labeler := &Labeler{}
		labeler.Add(attribute.String("server.key", "server.value"))

		ctx := ContextWithLabeler(context.Background(), labeler)
		got, ok := LabelerFromContext(ctx)
		assert.True(t, ok)
		assert.Same(t, labeler, got)
	})
}

func TestClientLabelerFromContext(t *testing.T) {
	t.Run("no client labeler in context", func(t *testing.T) {
		l, ok := ClientLabelerFromContext(context.Background())
		assert.False(t, ok)
		require.NotNil(t, l)
		assert.Empty(t, l.Get())
	})

	t.Run("client labeler set via ContextWithClientLabeler", func(t *testing.T) {
		labeler := &Labeler{}
		labeler.Add(attribute.String("client.key", "client.value"))

		ctx := ContextWithClientLabeler(context.Background(), labeler)
		got, ok := ClientLabelerFromContext(ctx)
		assert.True(t, ok)
		assert.Same(t, labeler, got)
	})
}

func TestClientLabelerContextIsolation(t *testing.T) {
	serverLabeler := &Labeler{}
	serverLabeler.Add(attribute.String("server.key", "server.value"))

	clientLabeler := &Labeler{}
	clientLabeler.Add(attribute.String("client.key", "client.value"))

	ctx := context.Background()
	ctx = ContextWithLabeler(ctx, serverLabeler)
	ctx = ContextWithClientLabeler(ctx, clientLabeler)

	gotServer, serverOK := LabelerFromContext(ctx)
	assert.True(t, serverOK)
	assert.Same(t, serverLabeler, gotServer)
	serverAttrs := gotServer.Get()
	require.Len(t, serverAttrs, 1)
	assert.Equal(t, attribute.String("server.key", "server.value"), serverAttrs[0])

	gotClient, clientOK := ClientLabelerFromContext(ctx)
	assert.True(t, clientOK)
	assert.Same(t, clientLabeler, gotClient)
	clientAttrs := gotClient.Get()
	require.Len(t, clientAttrs, 1)
	assert.Equal(t, attribute.String("client.key", "client.value"), clientAttrs[0])
}
