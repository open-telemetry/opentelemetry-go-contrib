// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSignerV4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"go.opentelemetry.io/otel/propagation"
)

type mockPropagator struct {
	injectKey   string
	injectValue string
}

func (p mockPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	carrier.Set(p.injectKey, p.injectValue)
}

func (p mockPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return context.TODO()
}

func (p mockPropagator) Fields() []string {
	return []string{}
}

func Test_otelMiddlewares_finalizeMiddlewareAfter(t *testing.T) {
	stack := middleware.Stack{
		Finalize: middleware.NewFinalizeStep(),
	}

	propagator := mockPropagator{
		injectKey:   "mock-key",
		injectValue: "mock-value",
	}

	m := otelMiddlewares{
		propagator: propagator,
	}

	err := m.finalizeMiddlewareAfter(&stack)
	require.NoError(t, err)

	input := &smithyhttp.Request{
		Request: &http.Request{
			Header: http.Header{},
		},
	}

	next := middleware.HandlerFunc(func(ctx context.Context, input interface{}) (output interface{}, metadata middleware.Metadata, err error) {
		return nil, middleware.Metadata{}, nil
	})

	_, _, err = stack.Finalize.HandleMiddleware(context.Background(), input, next)
	require.NoError(t, err)

	// Assert header has been updated with injected values
	key := http.CanonicalHeaderKey(propagator.injectKey)
	value := propagator.injectValue

	assert.Contains(t, input.Header, key)
	assert.Contains(t, input.Header[key], value)
}

func Test_otelMiddlewares_finalizeMiddlewareAfter_Noop(t *testing.T) {
	stack := middleware.Stack{
		Finalize: middleware.NewFinalizeStep(),
	}

	propagator := mockPropagator{
		injectKey:   "mock-key",
		injectValue: "mock-value",
	}

	m := otelMiddlewares{
		propagator: propagator,
	}

	err := m.finalizeMiddlewareAfter(&stack)
	require.NoError(t, err)

	// Non request input should trigger noop
	input := &struct{}{}

	next := middleware.HandlerFunc(func(ctx context.Context, input interface{}) (output interface{}, metadata middleware.Metadata, err error) {
		return nil, middleware.Metadata{}, nil
	})

	_, _, err = stack.Finalize.HandleMiddleware(context.Background(), input, next)
	assert.NoError(t, err)
}

type mockCredentialsProvider struct{}

func (mockCredentialsProvider) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{}, nil
}

type mockHTTPPresigner struct{}

func (f mockHTTPPresigner) PresignHTTP(
	ctx context.Context, credentials aws.Credentials, r *http.Request,
	payloadHash string, service string, region string, signingTime time.Time,
	optFns ...func(*awsSignerV4.SignerOptions),
) (
	url string, signedHeader http.Header, err error,
) {
	return "mock-url", nil, nil
}

func Test_otelMiddlewares_presignedRequests(t *testing.T) {
	stack := middleware.Stack{
		Finalize: middleware.NewFinalizeStep(),
	}

	presignedHTTPMiddleware := awsSignerV4.NewPresignHTTPRequestMiddleware(awsSignerV4.PresignHTTPRequestMiddlewareOptions{
		CredentialsProvider: mockCredentialsProvider{},
		Presigner:           mockHTTPPresigner{},
		LogSigning:          false,
	})

	err := stack.Finalize.Add(presignedHTTPMiddleware, middleware.After)
	require.NoError(t, err)

	propagator := mockPropagator{
		injectKey:   "mock-key",
		injectValue: "mock-value",
	}

	m := otelMiddlewares{
		propagator: propagator,
	}

	err = m.finalizeMiddlewareAfter(&stack)
	require.NoError(t, err)

	input := &smithyhttp.Request{
		Request: &http.Request{
			Header: http.Header{},
		},
	}

	next := middleware.HandlerFunc(func(ctx context.Context, input interface{}) (output interface{}, metadata middleware.Metadata, err error) {
		return nil, middleware.Metadata{}, nil
	})

	ctx := awsSignerV4.SetPayloadHash(context.Background(), "mock-hash")
	url, _, err := stack.Finalize.HandleMiddleware(ctx, input, next)

	// verify we actually went through the presign flow
	require.NoError(t, err)
	presignedReq, ok := url.(*awsSignerV4.PresignedHTTPRequest)
	require.True(t, ok)
	require.Equal(t, "mock-url", presignedReq.URL)

	// Assert header has NOT been updated with injected values, as the presign middleware should short circuit
	key := http.CanonicalHeaderKey(propagator.injectKey)
	value := propagator.injectValue

	assert.NotContains(t, input.Header, key)
	assert.NotContains(t, input.Header[key], value)
}

func Test_Span_name(t *testing.T) {
	serviceID1 := ""
	serviceID2 := "ServiceID"
	operation1 := ""
	operation2 := "Operation"

	assert.Equal(t, "", spanName(serviceID1, operation1))
	assert.Equal(t, spanName(serviceID1, operation2), "."+operation2)
	assert.Equal(t, spanName(serviceID2, operation1), serviceID2)
	assert.Equal(t, spanName(serviceID2, operation2), serviceID2+"."+operation2)
}
