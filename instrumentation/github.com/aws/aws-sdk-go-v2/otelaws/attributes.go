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
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
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
	return semconv.RPCSystemNameKey.String(AWSSystemVal)
}

// MethodAttr returns the RPC method attribute for AWS service and operation.
func MethodAttr(service, operation string) attribute.KeyValue {
	if service == "" {
		return semconv.RPCMethod(operation)
	}
	if operation == "" {
		return semconv.RPCMethod(service)
	}
	return semconv.RPCMethod(service + "/" + operation)
}

// OperationAttr returns the AWS operation attribute.
//
// Deprecated: use MethodAttr instead.
func OperationAttr(operation string) attribute.KeyValue {
	// rpc.service has been merged into rpc.method in semconv v1.39.0
	return MethodAttr("", operation)
}

// RegionAttr returns the AWS region attribute.
func RegionAttr(region string) attribute.KeyValue {
	return RegionKey.String(region)
}

// ServiceAttr returns the AWS service attribute.
//
// Deprecated: use MethodAttr instead.
func ServiceAttr(service string) attribute.KeyValue {
	// rpc.service has been merged into rpc.method in semconv v1.39.0
	return MethodAttr(service, "")
}

// RequestIDAttr returns the AWS request ID attribute.
func RequestIDAttr(requestID string) attribute.KeyValue {
	return RequestIDKey.String(requestID)
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
