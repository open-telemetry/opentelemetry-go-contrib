// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
)

// AWS attributes.
const (
	OperationKey attribute.Key = "aws.operation"
	RegionKey    attribute.Key = "aws.region"
	ServiceKey   attribute.Key = "aws.service"
	RequestIDKey attribute.Key = "aws.request_id"
)

var servicemap = map[string]AttributeSetter{
	dynamodb.ServiceID: DynamoDBAttributeSetter,
	sqs.ServiceID:      SQSAttributeSetter,
}

// OperationAttr returns the AWS operation attribute.
func OperationAttr(operation string) attribute.KeyValue {
	return OperationKey.String(operation)
}

// RegionAttr returns the AWS region attribute.
func RegionAttr(region string) attribute.KeyValue {
	return RegionKey.String(region)
}

// ServiceAttr returns the AWS service attribute.
func ServiceAttr(service string) attribute.KeyValue {
	return ServiceKey.String(service)
}

// RequestIDAttr returns the AWS request ID attribute.
func RequestIDAttr(requestID string) attribute.KeyValue {
	return RequestIDKey.String(requestID)
}

// DefaultAttributeSetter checks to see if there are service specific attributes available to set for the AWS service.
// If there are service specific attributes available then they will be included.
func DefaultAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	serviceID := v2Middleware.GetServiceID(ctx)

	if fn, ok := servicemap[serviceID]; ok {
		return fn(ctx, in)
	}

	return []attribute.KeyValue{}
}
