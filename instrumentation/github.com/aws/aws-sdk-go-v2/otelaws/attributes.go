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

package otelaws

import "go.opentelemetry.io/otel/attribute"

// AWS attributes.
const (
	OperationKey attribute.Key = "aws.operation"
	RegionKey    attribute.Key = "aws.region"
	ServiceKey   attribute.Key = "aws.service"
	RequestIDKey attribute.Key = "aws.request_id"
)

func OperationAttr(operation string) attribute.KeyValue {
	return OperationKey.String(operation)
}

func RegionAttr(region string) attribute.KeyValue {
	return RegionKey.String(region)
}

func ServiceAttr(service string) attribute.KeyValue {
	return ServiceKey.String(service)
}

func RequestIDAttr(requestID string) attribute.KeyValue {
	return RequestIDKey.String(requestID)
}
