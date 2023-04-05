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

package xray

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TestRemoteSamplerDescription assert remote sampling description.
func TestRemoteSamplerDescription(t *testing.T) {
	rs := &remoteSampler{}

	s := rs.Description()
	assert.Equal(t, s, "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}")
}

// assert that service name and cloud platform are obtained correctly from the resource.
func TestRemoteSamplerCreationWithPopulatedResource(t *testing.T) {
	endpoint, _ := url.Parse("http://127.0.0.1:2000")
	testResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		attribute.KeyValue{Key: semconv.ServiceNameKey, Value: attribute.StringValue("GOLANG_SAMPLING_TEST_SERVICE")},
		semconv.CloudPlatformAWSEC2,
	)

	rs, err := NewRemoteSamplerWithResource(context.TODO(), testResource, WithEndpoint(*endpoint), WithSamplingRulesPollingInterval(1000*time.Second))

	assert.Equal(t, rs.Description(), "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}")
	assert.NoError(t, err, "Failed to create a remote sampler with a resource populated with service name and cloud platform")
}

// assert that service name and cloud platform are set to empty string when not populated in the resource.
func TestRemoteSamplerCreationWithUnpopulatedResource(t *testing.T) {
	endpoint, _ := url.Parse("http://127.0.0.1:2000")
	testResource := resource.NewWithAttributes(semconv.SchemaURL)

	rs, err := NewRemoteSamplerWithResource(context.TODO(), testResource, WithEndpoint(*endpoint), WithSamplingRulesPollingInterval(1000*time.Second))

	assert.Equal(t, rs.Description(), "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}")
	assert.NoError(t, err, "Failed to create a remote sampler with an unpopulated resource")
}
