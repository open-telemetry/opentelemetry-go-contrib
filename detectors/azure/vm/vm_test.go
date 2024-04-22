// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func Test_Detect(t *testing.T) {
	type input struct {
		jsonMetadata string
		err          error
	}
	type expected struct {
		resource *resource.Resource
		err      bool
	}
	type testCase struct {
		input    input
		expected expected
	}

	testTable := []testCase{
		{
			input: input{
				jsonMetadata: `{ 
					"location": "us-west3",
					"resourceId": "/subscriptions/sid/resourceGroups/rid/providers/pname/name",
					"vmId": "43f65c49-8715-4639-88a9-be6d7eb749a5",
					"name": "localhost-3",
					"vmSize": "Standard_D2s_v3",
					"osType": "linux",
					"version": "6.5.0-26-generic"
				}`,
				err: nil,
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					semconv.CloudProviderAzure,
					semconv.CloudPlatformAzureVM,
					semconv.CloudRegion("us-west3"),
					semconv.CloudResourceID("/subscriptions/sid/resourceGroups/rid/providers/pname/name"),
					semconv.HostID("43f65c49-8715-4639-88a9-be6d7eb749a5"),
					semconv.HostName("localhost-3"),
					semconv.HostType("Standard_D2s_v3"),
					semconv.OSTypeKey.String("linux"),
					semconv.OSVersion("6.5.0-26-generic"),
				}...),
				err: false,
			},
		},
		{
			input: input{
				jsonMetadata: `{`,
				err:          nil,
			},
			expected: expected{
				resource: nil,
				err:      true,
			},
		},
		{
			input: input{
				jsonMetadata: "",
				err:          errors.New("cannot get metadata"),
			},
			expected: expected{
				resource: nil,
				err:      true,
			},
		},
	}

	for _, tCase := range testTable {
		detector := NewResourceDetector(WithClient(&mockClient{
			jsonMetadata: []byte(tCase.input.jsonMetadata),
			err:          tCase.input.err,
		}))

		azureResource, err := detector.Detect(context.Background())

		assert.Equal(t, err != nil, tCase.expected.err)
		assert.Equal(t, tCase.expected.resource, azureResource)
	}
}

type mockClient struct {
	jsonMetadata []byte
	err          error
}

func (c *mockClient) GetJSONMetadata() ([]byte, error) {
	return c.jsonMetadata, c.err
}
