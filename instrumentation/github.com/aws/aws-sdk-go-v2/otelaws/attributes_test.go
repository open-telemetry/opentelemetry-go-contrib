// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"

	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func TestOperationAttr(t *testing.T) {
	operation := "test-operation"
	attr := OperationAttr(operation)
	assert.Equal(t, attribute.String("rpc.method", operation), attr)
}

func TestRegionAttr(t *testing.T) {
	region := "test-region"
	attr := RegionAttr(region)
	assert.Equal(t, attribute.String("aws.region", region), attr)
}

func TestServiceAttr(t *testing.T) {
	service := "test-service"
	attr := ServiceAttr(service)
	assert.Equal(t, semconv.RPCService(service), attr)
}

func TestRequestIDAttr(t *testing.T) {
	requestID := "test-request-id"
	attr := RequestIDAttr(requestID)
	assert.Equal(t, attribute.String("aws.request_id", requestID), attr)
}

func TestSystemAttribute(t *testing.T) {
	attr := SystemAttr()
	assert.Equal(t, semconv.RPCSystemKey.String("aws-api"), attr)
}

func TestDefaultAttributeBuilderNotSupportedService(t *testing.T) {
	testCtx := awsMiddleware.SetServiceID(context.TODO(), "not-implemented-service")

	attr := DefaultAttributeBuilder(testCtx, middleware.InitializeInput{}, middleware.InitializeOutput{})
	assert.Empty(t, attr)
}

func TestDefaultAttributeBuilderOnSupportedService(t *testing.T) {
	testCtx := awsMiddleware.SetServiceID(context.TODO(), sqs.ServiceID)
	testQueueURL := "test-queue-url"

	attr := DefaultAttributeBuilder(testCtx, middleware.InitializeInput{
		Parameters: &sqs.SendMessageInput{
			QueueUrl: &testQueueURL,
		},
	}, middleware.InitializeOutput{})
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.MessagingSystem("AmazonSQS"),
		semconv.NetPeerName(testQueueURL),
	}, attr)
}
