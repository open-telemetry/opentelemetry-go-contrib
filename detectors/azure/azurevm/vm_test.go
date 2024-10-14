// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurevm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestDetect(t *testing.T) {
	type input struct {
		jsonMetadata string
		statusCode   int
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
				statusCode: http.StatusOK,
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
				statusCode:   http.StatusOK,
			},
			expected: expected{
				resource: nil,
				err:      true,
			},
		},
		{
			input: input{
				jsonMetadata: "",
				statusCode:   http.StatusNotFound,
			},
			expected: expected{
				resource: resource.Empty(),
				err:      false,
			},
		},
		{
			input: input{
				jsonMetadata: "",
				statusCode:   http.StatusInternalServerError,
			},
			expected: expected{
				resource: nil,
				err:      true,
			},
		},
	}

	for _, tCase := range testTable {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tCase.input.statusCode)

			if r.Header.Get("Metadata") == "True" {
				fmt.Fprint(w, tCase.input.jsonMetadata)
			}
		}))

		detector := New()
		detector.endpoint = svr.URL

		azureResource, err := detector.Detect(context.Background())

		svr.Close()

		assert.Equal(t, err != nil, tCase.expected.err)
		assert.Equal(t, tCase.expected.resource, azureResource)
	}
}
