// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// successfully return resource when process is running on Amazon Lambda environment.
func TestDetectSuccess(t *testing.T) {
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogGroupNameEnvVar, "/aws/lambda/testFunction")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSLambda,
		semconv.FaaSName("testFunction"),
		semconv.CloudRegion("us-texas-1"),
		semconv.FaaSVersion("$LATEST"),
		semconv.FaaSInstance("2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"),
		semconv.AWSLogStreamNames("2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc"),
		semconv.AWSLogGroupNames("/aws/lambda/testFunction"),
		semconv.FaaSMaxMemory(128 * miB),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := NewResourceDetector()
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// return empty resource and no error when not running on lambda.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := NewResourceDetector()
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error when not running on Lambda")
	assert.Empty(t, res.Attributes())
}

// only emit attributes whose environment variable is actually set: an unset
// variable must not surface as an empty-string attribute.
func TestDetectPartialEnv(t *testing.T) {
	os.Clearenv()
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")

	detector := NewResourceDetector()
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSLambda,
		semconv.FaaSName("testFunction"),
	}, res.Attributes())
}

// an unparsable memory limit is skipped rather than reported as zero.
func TestDetectInvalidMemoryLimit(t *testing.T) {
	os.Clearenv()
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(lambdaMemoryLimitEnvVar, "not-a-number")

	detector := NewResourceDetector()
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	for _, kv := range res.Attributes() {
		assert.NotEqual(t, semconv.FaaSMaxMemoryKey, kv.Key)
	}
}

func TestDetectWithAttributeFilter(t *testing.T) {
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogGroupNameEnvVar, "/aws/lambda/testFunction")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")

	detector := NewResourceDetector(WithAttributeFilter(func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.FaaSNameKey
	}))
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, []attribute.KeyValue{semconv.FaaSName("testFunction")}, res.Attributes())
}
