// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ec2

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

type mockClient struct {
	mock.Mock
}

func (m *mockClient) GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*imds.GetInstanceIdentityDocumentOutput), args.Error(1)
}

func (m *mockClient) GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*imds.GetMetadataOutput), args.Error(1)
}

type testCase struct {
	name           string
	metadataOutput *imds.GetMetadataOutput
	metadataErr    error
	docOutput      *imds.GetInstanceIdentityDocumentOutput
	docErr         error
	expectedAttrs  []attribute.KeyValue
	expectedErr    error
}

func TestAWSResourceDetection(t *testing.T) {
	doc := validIdentityDocument()

	testCases := []testCase{
		{
			name:           "AllFields",
			docOutput:      doc,
			metadataOutput: mockMetadataOutput("ip-12-34-56-78.us-west-2.compute.internal"),
			expectedAttrs: []attribute.KeyValue{
				semconv.CloudProviderAWS,
				semconv.CloudPlatformAWSEC2,
				semconv.CloudRegion("us-west-2"),
				semconv.CloudAvailabilityZone("us-west-2b"),
				semconv.CloudAccountID("123456789012"),
				semconv.HostID("i-1234567890abcdef0"),
				semconv.HostImageID("ami-5fb8c835"),
				semconv.HostType("t2.micro"),
				semconv.HostName("ip-12-34-56-78.us-west-2.compute.internal"),
			},
		},
		{
			name:           "NoHostname",
			docOutput:      doc,
			metadataOutput: mockMetadataOutput(""),
			metadataErr:    errors.New("mock error"),
			expectedAttrs: []attribute.KeyValue{
				semconv.CloudProviderAWS,
				semconv.CloudPlatformAWSEC2,
				semconv.CloudRegion("us-west-2"),
				semconv.CloudAvailabilityZone("us-west-2b"),
				semconv.CloudAccountID("123456789012"),
				semconv.HostID("i-1234567890abcdef0"),
				semconv.HostImageID("ami-5fb8c835"),
				semconv.HostType("t2.micro"),
			},
		},
		{
			name:           "NonEC2Host",
			docErr:         errors.New("error getting InstanceIdentityDocument"),
			docOutput:      &imds.GetInstanceIdentityDocumentOutput{},
			metadataOutput: mockMetadataOutput(""),
			expectedAttrs:  nil, // Empty resource
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientMock := new(mockClient)

			clientMock.On("GetInstanceIdentityDocument", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.docOutput, tc.docErr)
			clientMock.On("GetMetadata", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.metadataOutput, tc.metadataErr)

			detector := &resourceDetector{c: clientMock}
			res, _ := detector.Detect(t.Context())

			if tc.expectedAttrs == nil {
				assert.Equal(t, resource.Empty(), res, "Resource should be empty")
			} else {
				expected := resource.NewWithAttributes(semconv.SchemaURL, tc.expectedAttrs...)
				assert.Equal(t, expected, res, "Resource returned is incorrect")
			}
		})
	}
}

func TestAWSInvalidClient(t *testing.T) {
	detector := &resourceDetector{c: nil}
	_, err := detector.Detect(t.Context())
	assert.ErrorIs(t, err, errClient)
}

func TestRecordErrors(t *testing.T) {
	doc := validIdentityDocument()

	testCases := []testCase{
		{
			name:        "404 returns no error",
			docOutput:   doc,
			metadataErr: newAwsResponseError(404),
		},
		{
			name:        "502 returns error",
			docOutput:   doc,
			metadataErr: newAwsResponseError(502),
			expectedErr: resource.ErrPartialResource,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientMock := new(mockClient)

			clientMock.On("GetInstanceIdentityDocument", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.docOutput, tc.docErr)
			clientMock.On("GetMetadata", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.metadataOutput, tc.metadataErr)

			detector := &resourceDetector{c: clientMock}
			_, err := detector.Detect(t.Context())
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func validIdentityDocument() *imds.GetInstanceIdentityDocumentOutput {
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
		ResultMetadata:           middleware.Metadata{},
	}
}

func mockMetadataOutput(val string) *imds.GetMetadataOutput {
	return &imds.GetMetadataOutput{
		Content: io.NopCloser(bytes.NewReader([]byte(val))),
	}
}

func newAwsResponseError(statusCode int) *awshttp.ResponseError {
	err := &smithyhttp.ResponseError{
		Response: &smithyhttp.Response{
			Response: &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader("Bad Request")),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			},
		},
		Err: errors.New("error fetching metadata"),
	}

	return &awshttp.ResponseError{
		ResponseError: err,
		RequestID:     "test123",
	}
}
