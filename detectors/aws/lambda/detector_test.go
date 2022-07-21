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

package lambda

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// successfully return resource when process is running on Amazon Lambda environment.
func TestDetectSuccess(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	_ = os.Setenv(awsRegionEnvVar, "us-texas-1")
	_ = os.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudRegionKey.String("us-texas-1"),
		semconv.FaaSNameKey.String("testFunction"),
		semconv.FaaSVersionKey.String("$LATEST"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := resourceDetector{}
	res, err := detector.Detect(context.Background())

	assert.Nil(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// return empty resource when not running on lambda.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := resourceDetector{}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errNotOnLambda, err)
	assert.Equal(t, 0, len(res.Attributes()))
}
