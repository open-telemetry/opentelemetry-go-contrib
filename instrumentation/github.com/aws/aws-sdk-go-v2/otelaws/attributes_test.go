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

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestOperationAttr(t *testing.T) {
	operation := "test-operation"
	attr := OperationAttr(operation)
	assert.Equal(t, attribute.String("aws.operation", operation), attr)
}

func TestRegionAttr(t *testing.T) {
	region := "test-region"
	attr := RegionAttr(region)
	assert.Equal(t, attribute.String("aws.region", region), attr)
}

func TestServiceAttr(t *testing.T) {
	service := "test-service"
	attr := ServiceAttr(service)
	assert.Equal(t, attribute.String("aws.service", service), attr)
}

func TestRequestIDAttr(t *testing.T) {
	requestID := "test-request-id"
	attr := RequestIDAttr(requestID)
	assert.Equal(t, attribute.String("aws.request_id", requestID), attr)
}
