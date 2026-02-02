// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"testing"

	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
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
	assert.Equal(t, semconv.RPCMethod(service), attr)
}

func TestRequestIDAttr(t *testing.T) {
	requestID := "test-request-id"
	attr := RequestIDAttr(requestID)
	assert.Equal(t, attribute.String("aws.request_id", requestID), attr)
}

func TestSystemAttribute(t *testing.T) {
	attr := SystemAttr()
	assert.Equal(t, semconv.RPCSystemNameKey.String("aws-api"), attr)
}

func TestMethodAttr(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		operation string
		want      attribute.KeyValue
	}{
		{
			name:      "both service and operation",
			service:   "DynamoDB",
			operation: "GetItem",
			want:      attribute.String("rpc.method", "DynamoDB/GetItem"),
		},
		{
			name:      "service only",
			service:   "Route 53",
			operation: "",
			want:      attribute.String("rpc.method", "Route 53"),
		},
		{
			name:      "operation only",
			service:   "",
			operation: "DescribeInstances",
			want:      attribute.String("rpc.method", "DescribeInstances"),
		},
		{
			name:      "both empty",
			service:   "",
			operation: "",
			want:      attribute.String("rpc.method", ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := MethodAttr(tt.service, tt.operation)
			assert.Equal(t, tt.want, attr)
		})
	}
}

func TestDefaultAttributeBuilderNotSupportedService(t *testing.T) {
	testCtx := awsMiddleware.SetServiceID(t.Context(), "not-implemented-service")

	attr := DefaultAttributeBuilder(testCtx, middleware.InitializeInput{}, middleware.InitializeOutput{})
	assert.Empty(t, attr)
}

func TestDefaultAttributeBuilderOnSupportedService(t *testing.T) {
	testCtx := awsMiddleware.SetServiceID(t.Context(), sqs.ServiceID)
	testQueueURL := "test-queue-url"

	attr := DefaultAttributeBuilder(testCtx, middleware.InitializeInput{
		Parameters: &sqs.SendMessageInput{
			QueueUrl: &testQueueURL,
		},
	}, middleware.InitializeOutput{})
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.MessagingSystemAWSSQS,
		semconv.ServerAddress(testQueueURL),
	}, attr)
}
