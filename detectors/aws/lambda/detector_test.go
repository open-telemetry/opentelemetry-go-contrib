// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// successfully return resource when process is running on Amazon Lambda environment.
func TestDetectSuccess(t *testing.T) {
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudRegion("us-texas-1"),
		semconv.FaaSName("testFunction"),
		semconv.FaaSVersion("$LATEST"),
		semconv.FaaSInstance("2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"),
		semconv.FaaSMaxMemory(128),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := resourceDetector{}
	res, err := detector.Detect(context.Background())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// return empty resource when not running on lambda.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := resourceDetector{}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errNotOnLambda, err)
	assert.Empty(t, res.Attributes())
}
