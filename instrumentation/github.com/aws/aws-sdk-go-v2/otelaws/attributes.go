// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// AWS attributes.
const (
	RegionKey    attribute.Key = "aws.region"
	RequestIDKey attribute.Key = "aws.request_id"
	AWSSystemVal string        = "aws-api"
)

var servicemap = map[string]AttributeBuilder{
	dynamodb.ServiceID: DynamoDBAttributeBuilder,
	sqs.ServiceID:      SQSAttributeBuilder,
	sns.ServiceID:      SNSAttributeBuilder,
}

// SystemAttr return the AWS RPC system attribute.
func SystemAttr() attribute.KeyValue {
	return semconv.RPCSystemKey.String(AWSSystemVal)
}

// OperationAttr returns the AWS operation attribute.
func OperationAttr(operation string) attribute.KeyValue {
	return semconv.RPCMethod(operation)
}

// RegionAttr returns the AWS region attribute.
func RegionAttr(region string) attribute.KeyValue {
	return RegionKey.String(region)
}

// ServiceAttr returns the AWS service attribute.
func ServiceAttr(service string) attribute.KeyValue {
	return semconv.RPCService(service)
}

// RequestIDAttr returns the AWS request ID attribute.
func RequestIDAttr(requestID string) attribute.KeyValue {
	return RequestIDKey.String(requestID)
}

// DefaultAttributeSetter checks to see if there are service specific attributes available to set for the AWS service.
// If there are service specific attributes available then they will be included.
//
// Deprecated: Use DefaultAttributeBuilder instead. This will be removed in a future release.
func DefaultAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	return DefaultAttributeBuilder(ctx, in, middleware.InitializeOutput{})
}

// DefaultAttributeBuilder checks to see if there are service specific attributes available to set for the AWS service.
// If there are service specific attributes available then they will be included.
func DefaultAttributeBuilder(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue {
	serviceID := v2Middleware.GetServiceID(ctx)

	if fn, ok := servicemap[serviceID]; ok {
		return fn(ctx, in, out)
	}

	return []attribute.KeyValue{}
}
