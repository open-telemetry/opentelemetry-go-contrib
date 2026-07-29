// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurevm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// fullMetadata exercises every emitted attribute, including the computerName
// host.name source and the tag list.
const fullMetadata = `{
	"location": "us-west3",
	"resourceId": "/subscriptions/sid/resourceGroups/rid/providers/pname/name",
	"vmId": "43f65c49-8715-4639-88a9-be6d7eb749a5",
	"name": "localhost-3",
	"vmSize": "Standard_D2s_v3",
	"osType": "linux",
	"version": "6.5.0-26-generic",
	"subscriptionId": "sid",
	"resourceGroupName": "rid",
	"vmScaleSetName": "myScaleset",
	"zone": "1",
	"osProfile": { "computerName": "computer-name" },
	"tagsList": [
		{ "name": "env", "value": "prod" },
		{ "name": "team", "value": "obs" },
		{ "name": "secret", "value": "hidden" }
	]
}`

func TestNew(t *testing.T) {
	// New must remain equivalent to NewResourceDetector called with no options.
	assert.Equal(t, NewResourceDetector(), New())
	assert.Equal(t, defaultAzureVMMetadataEndpoint, New().endpoint)
}

func TestDetect(t *testing.T) {
	type input struct {
		jsonMetadata string
		statusCode   int
		opts         []Option
	}
	type expected struct {
		resource *resource.Resource
		err      bool
	}
	type testCase struct {
		name     string
		input    input
		expected expected
	}

	testTable := []testCase{
		{
			name: "full metadata with tag key filter",
			input: input{
				jsonMetadata: fullMetadata,
				statusCode:   http.StatusOK,
				opts:         []Option{WithTagKeyFilter(regexp.MustCompile("^(env|team)$").MatchString)},
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					semconv.CloudProviderAzure,
					semconv.CloudPlatformAzureVM,
					semconv.CloudRegion("us-west3"),
					semconv.CloudResourceID("/subscriptions/sid/resourceGroups/rid/providers/pname/name"),
					semconv.HostID("43f65c49-8715-4639-88a9-be6d7eb749a5"),
					semconv.HostName("computer-name"),
					semconv.HostType("Standard_D2s_v3"),
					semconv.OSTypeKey.String("linux"),
					semconv.OSVersion("6.5.0-26-generic"),
					semconv.CloudAccountID("sid"),
					semconv.CloudAvailabilityZone("1"),
					azureVMNameKey.String("localhost-3"),
					azureVMSizeKey.String("Standard_D2s_v3"),
					azureVMScaleSetNameKey.String("myScaleset"),
					semconv.AzureResourceGroupName("rid"),
					attribute.String("azure.tag.env", "prod"),
					attribute.String("azure.tag.team", "obs"),
				}...),
				err: false,
			},
		},
		{
			name: "host.name falls back to the VM name",
			input: input{
				// computerName is absent and the optional fields (zone,
				// scaleset, tags) are omitted.
				jsonMetadata: `{
					"location": "us-west3",
					"vmId": "43f65c49-8715-4639-88a9-be6d7eb749a5",
					"name": "localhost-3",
					"vmSize": "Standard_D2s_v3",
					"subscriptionId": "sid",
					"resourceGroupName": "rid"
				}`,
				statusCode: http.StatusOK,
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					semconv.CloudProviderAzure,
					semconv.CloudPlatformAzureVM,
					semconv.CloudRegion("us-west3"),
					semconv.HostID("43f65c49-8715-4639-88a9-be6d7eb749a5"),
					semconv.HostName("localhost-3"),
					semconv.HostType("Standard_D2s_v3"),
					semconv.CloudAccountID("sid"),
					azureVMNameKey.String("localhost-3"),
					azureVMSizeKey.String("Standard_D2s_v3"),
					semconv.AzureResourceGroupName("rid"),
				}...),
				err: false,
			},
		},
		{
			name: "tags are not emitted without a tag key filter",
			input: input{
				jsonMetadata: `{
					"location": "us-west3",
					"vmId": "43f65c49-8715-4639-88a9-be6d7eb749a5",
					"name": "localhost-3",
					"tagsList": [ { "name": "env", "value": "prod" } ]
				}`,
				statusCode: http.StatusOK,
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					semconv.CloudProviderAzure,
					semconv.CloudPlatformAzureVM,
					semconv.CloudRegion("us-west3"),
					semconv.HostID("43f65c49-8715-4639-88a9-be6d7eb749a5"),
					semconv.HostName("localhost-3"),
					azureVMNameKey.String("localhost-3"),
				}...),
				err: false,
			},
		},
		{
			name: "attribute filter",
			input: input{
				jsonMetadata: fullMetadata,
				statusCode:   http.StatusOK,
				opts: []Option{WithAttributeFilter(func(kv attribute.KeyValue) bool {
					return strings.HasPrefix(string(kv.Key), "cloud.")
				})},
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					semconv.CloudProviderAzure,
					semconv.CloudPlatformAzureVM,
					semconv.CloudRegion("us-west3"),
					semconv.CloudResourceID("/subscriptions/sid/resourceGroups/rid/providers/pname/name"),
					semconv.CloudAccountID("sid"),
					semconv.CloudAvailabilityZone("1"),
				}...),
				err: false,
			},
		},
		{
			name: "attribute filter applies to tag attributes",
			input: input{
				// The tag key filter selects which tags become attributes; the
				// attribute filter is applied last, over those attributes too.
				jsonMetadata: fullMetadata,
				statusCode:   http.StatusOK,
				opts: []Option{
					WithTagKeyFilter(regexp.MustCompile("^(env|team)$").MatchString),
					WithAttributeFilter(func(kv attribute.KeyValue) bool {
						return strings.HasPrefix(string(kv.Key), "azure.") &&
							!strings.HasPrefix(string(kv.Key), azureTagPrefix)
					}),
				},
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
					azureVMNameKey.String("localhost-3"),
					azureVMSizeKey.String("Standard_D2s_v3"),
					azureVMScaleSetNameKey.String("myScaleset"),
					semconv.AzureResourceGroupName("rid"),
				}...),
				err: false,
			},
		},
		{
			name: "attribute filter rejecting everything",
			input: input{
				jsonMetadata: fullMetadata,
				statusCode:   http.StatusOK,
				opts: []Option{
					WithTagKeyFilter(func(string) bool { return true }),
					WithAttributeFilter(func(attribute.KeyValue) bool { return false }),
				},
			},
			expected: expected{
				resource: resource.NewWithAttributes(semconv.SchemaURL),
				err:      false,
			},
		},
		{
			name: "malformed metadata",
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
			name: "not running in Azure",
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
			name: "metadata endpoint failure",
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
		t.Run(tCase.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tCase.input.statusCode)

				if r.Header.Get("Metadata") == "True" {
					fmt.Fprint(w, tCase.input.jsonMetadata)
				}
			}))
			defer svr.Close()

			detector := NewResourceDetector(tCase.input.opts...)
			detector.endpoint = svr.URL

			azureResource, err := detector.Detect(t.Context())

			assert.Equal(t, tCase.expected.err, err != nil)
			assert.Equal(t, tCase.expected.resource, azureResource)
		})
	}
}
