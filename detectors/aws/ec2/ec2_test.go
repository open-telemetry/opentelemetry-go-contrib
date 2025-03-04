// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ec2

import (
	"bytes"
	"context"
	"errors"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestAWS_Detect(t *testing.T) {
	type fields struct {
		Client Client
	}

	type want struct {
		Error    string
		Partial  bool
		Resource *resource.Resource
	}

	usWestInst := func() (*imds.GetInstanceIdentityDocumentOutput, error) {
		// Example from https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
		doc := imds.InstanceIdentityDocument{
			MarketplaceProductCodes: []string{"1abc2defghijklm3nopqrs4tu"},
			AvailabilityZone:        "us-west-2b",
			PrivateIP:               "10.158.112.84",
			Version:                 "2017-09-30",
			Region:                  "us-west-2",
			InstanceID:              "i-1234567890abcdef0",
			InstanceType:            "t2.micro",
			AccountID:               "123456789012",
			PendingTime:             time.Date(2016, time.November, 19, 16, 32, 11, 0, time.UTC),
			ImageID:                 "ami-5fb8c835",
			Architecture:            "x86_64",
		}

		return &imds.GetInstanceIdentityDocumentOutput{
			InstanceIdentityDocument: doc,
		}, nil

	}

	usWestIDLabels := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEC2,
		semconv.CloudRegion("us-west-2"),
		semconv.CloudAvailabilityZone("us-west-2b"),
		semconv.CloudAccountID("123456789012"),
		semconv.HostID("i-1234567890abcdef0"),
		semconv.HostImageID("ami-5fb8c835"),
		semconv.HostType("t2.micro"),
	}

	testTable := map[string]struct {
		Fields fields
		Want   want
	}{
		"Instance ID Error": {
			Fields: fields{
				Client: &clientMock{idDoc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return &imds.GetInstanceIdentityDocumentOutput{}, errors.New("id not available")
				}},
			},
			Want: want{Error: "id not available"},
		},
		"Hostname Not Found": {
			Fields: fields{
				Client: &clientMock{idDoc: usWestInst, metadata: map[string]meta{}},
			},
			Want: want{Resource: resource.NewWithAttributes(semconv.SchemaURL, usWestIDLabels...)},
		},
		"Hostname Response Error": {
			Fields: fields{
				Client: &clientMock{
					idDoc: usWestInst,
					metadata: map[string]meta{
						"hostname": {
							err: &awshttp.ResponseError{
								ResponseError: &smithyhttp.ResponseError{
									Err: errors.New("EC2MetadataError"),
									Response: &smithyhttp.Response{
										Response: &http.Response{
											StatusCode: 500,
										},
									},
								},
								RequestID: "test-request",
							},
						},
					},
				},
			},
			Want: want{
				Error:    `partial resource: ["hostname": 500 https response error StatusCode: 500, RequestID: test-request, EC2MetadataError]`,
				Partial:  true,
				Resource: resource.NewWithAttributes(semconv.SchemaURL, usWestIDLabels...),
			},
		},
		"Hostname General Error": {
			Fields: fields{
				Client: &clientMock{
					idDoc: usWestInst,
					metadata: map[string]meta{
						"hostname": {err: errors.New("unknown error")},
					},
				},
			},
			Want: want{
				Error:    `partial resource: ["hostname": unknown error]`,
				Partial:  true,
				Resource: resource.NewWithAttributes(semconv.SchemaURL, usWestIDLabels...),
			},
		},
		"All Available": {
			Fields: fields{
				Client: &clientMock{
					idDoc: usWestInst,
					metadata: map[string]meta{
						"hostname": {value: "ip-12-34-56-78.us-west-2.compute.internal"},
					},
				},
			},
			Want: want{Resource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderAWS,
				semconv.CloudPlatformAWSEC2,
				semconv.CloudRegion("us-west-2"),
				semconv.CloudAvailabilityZone("us-west-2b"),
				semconv.CloudAccountID("123456789012"),
				semconv.HostID("i-1234567890abcdef0"),
				semconv.HostImageID("ami-5fb8c835"),
				semconv.HostName("ip-12-34-56-78.us-west-2.compute.internal"),
				semconv.HostType("t2.micro"),
			)},
		},
	}

	for name, tt := range testTable {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ec2ResourceDetector := NewResourceDetector(WithClient(tt.Fields.Client))

			r, err := ec2ResourceDetector.Detect(context.Background())

			assert.Equal(t, tt.Want.Resource, r, "Resource")

			if tt.Want.Error != "" {
				require.EqualError(t, err, tt.Want.Error, "Error")
				assert.Equal(t, tt.Want.Partial, errors.Is(err, resource.ErrPartialResource), "Partial Resource")
				return
			}

			require.NoError(t, err, "Error")
		})
	}
}

type clientMock struct {
	idDoc    func() (*imds.GetInstanceIdentityDocumentOutput, error)
	metadata map[string]meta
}

type meta struct {
	err   error
	value string
}

func (c *clientMock) GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	return c.idDoc()
}

func (c *clientMock) GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error) {
	v, ok := c.metadata[params.Path]
	if !ok {
		return nil, createEC2MetadataError()
	}

	if v.err != nil {
		return nil, v.err
	}

	return &imds.GetMetadataOutput{
		Content: io.NopCloser(bytes.NewReader([]byte(v.value))),
	}, nil

}

func createEC2MetadataError() error {
	smithyResp := &smithyhttp.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
		},
	}

	smithyRespErr := &smithyhttp.ResponseError{
		Response: smithyResp,
		Err: &smithy.GenericAPIError{
			Code:    "EC2MetadataError",
			Message: "failed to make EC2Metadata request",
			Fault:   smithy.FaultClient,
		},
	}

	return &awshttp.ResponseError{
		ResponseError: smithyRespErr,
		RequestID:     "test-request",
	}
}
