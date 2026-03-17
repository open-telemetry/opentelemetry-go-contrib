// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
)

func TestSDKResource(t *testing.T) {
	t.Run("returns nil for zero value sdk", func(t *testing.T) {
		var sdk SDK
		assert.Nil(t, sdk.Resource())
	})

	t.Run("returns resource even when providers are not configured", func(t *testing.T) {
		sdk, err := NewSDK(
			WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
				Resource: &Resource{
					Attributes: []AttributeNameValue{
						{Name: "service.name", Value: "collector"},
					},
				},
			}),
		)
		assert.NoError(t, err)

		res := sdk.Resource()
		assert.NotNil(t, res)
		assert.Contains(t, res.Attributes(), attribute.String("service.name", "collector"))
	})

	t.Run("returns empty resource for disabled sdk", func(t *testing.T) {
		sdk, err := NewSDK(
			WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
				Disabled: ptr(true),
			}),
		)
		assert.NoError(t, err)
		assert.NotNil(t, sdk.Resource())
		assert.Empty(t, sdk.Resource().Attributes())
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		sdk, err := NewSDK(
			WithOpenTelemetryConfiguration(OpenTelemetryConfiguration{
				Resource: &Resource{
					Attributes: []AttributeNameValue{
						{Name: "service.name", Value: "collector"},
					},
				},
			}),
		)
		assert.NoError(t, err)

		res := sdk.Resource()
		assert.NotNil(t, res)

		newRes := sdkresource.NewSchemaless(attribute.String("service.name", "mutated"))
		*res = *newRes

		assert.Contains(t, res.Attributes(), attribute.String("service.name", "mutated"))
		assert.Contains(t, sdk.Resource().Attributes(), attribute.String("service.name", "collector"))
		assert.NotContains(t, sdk.Resource().Attributes(), attribute.String("service.name", "mutated"))
	})
}
