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

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
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

			res, _ := detectWithClient(t, clientMock)

			if tc.expectedAttrs == nil {
				assert.Equal(t, resource.Empty(), res, "Resource should be empty")
			} else {
				expected := resource.NewWithAttributes(semconv.SchemaURL, tc.expectedAttrs...)
				assert.Equal(t, expected, res, "Resource returned is incorrect")
			}
		})
	}
}

func TestNewResourceDetector(t *testing.T) {
	assert.NotNil(t, NewResourceDetector())
}

func TestNewResourceDetectorWithOptions(t *testing.T) {
	logger := logging.NewStandardLogger(io.Discard)

	testCases := []struct {
		name           string
		opts           []Option
		expectedLogger logging.Logger
	}{
		{
			name: "no options",
		},
		{
			name:           "with logger",
			opts:           []Option{WithAWSLogger(logger)},
			expectedLogger: logger,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detector := NewResourceDetectorWithOptions(tc.opts...)
			require.NotNil(t, detector)
			assert.Equal(t, tc.expectedLogger, detector.(*resourceDetector).cfg.logger)
		})
	}
}

// TestDetectReturnsClientError verifies the client construction error is
// surfaced from Detect rather than the constructor.
func TestDetectReturnsClientError(t *testing.T) {
	errLoad := errors.New("load config failed")
	stubClientSeams(
		t,
		func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, errLoad
		},
		func(aws.Config) client {
			t.Fatal("newIMDSClient should not be called")
			return nil
		},
	)

	_, err := NewResourceDetector().Detect(t.Context())
	assert.ErrorIs(t, err, errLoad)
}

func TestNewClient(t *testing.T) {
	t.Run("returns error when config load fails", func(t *testing.T) {
		stubClientSeams(
			t,
			func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
				return aws.Config{}, errors.New("load failed")
			},
			func(aws.Config) client {
				t.Fatal("newIMDSClient should not be called")
				return nil
			},
		)

		c, err := newClient(config{})
		require.Error(t, err)
		assert.Nil(t, c)
	})

	t.Run("passes configured logger to the AWS config", func(t *testing.T) {
		logger := logging.NewStandardLogger(io.Discard)
		var actualLogger logging.Logger
		stubClientSeams(
			t,
			func(_ context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
				var lo awsconfig.LoadOptions
				for _, fn := range optFns {
					require.NoError(t, fn(&lo))
				}
				actualLogger = lo.Logger
				return aws.Config{}, nil
			},
			func(aws.Config) client { return new(mockClient) },
		)

		_, err := newClient(config{logger: logger})
		require.NoError(t, err)
		assert.Same(t, logger, actualLogger)
	})

	t.Run("returns created imds client when config load succeeds", func(t *testing.T) {
		fakeClient := new(mockClient)
		stubClientSeams(
			t,
			func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
				return aws.Config{Region: "us-west-2"}, nil
			},
			func(cfg aws.Config) client {
				assert.Equal(t, "us-west-2", cfg.Region)
				return fakeClient
			},
		)

		c, err := newClient(config{})
		require.NoError(t, err)
		assert.Same(t, fakeClient, c)
	})
}

// stubClientSeams temporarily replaces the AWS client construction seams,
// restoring them when the test completes.
func stubClientSeams(
	t *testing.T,
	load func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error),
	newIMDS func(aws.Config) client,
) {
	t.Helper()
	origLoad, origNew := loadDefaultConfig, newIMDSClient
	t.Cleanup(func() {
		loadDefaultConfig = origLoad
		newIMDSClient = origNew
	})
	loadDefaultConfig = load
	newIMDSClient = newIMDS
}

// detectWithClient runs Detect with c injected as the IMDS client via the
// construction seams.
func detectWithClient(t *testing.T, c client) (*resource.Resource, error) {
	t.Helper()
	stubClientSeams(
		t,
		func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		},
		func(aws.Config) client { return c },
	)
	return NewResourceDetector().Detect(t.Context())
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

			_, err := detectWithClient(t, clientMock)
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
