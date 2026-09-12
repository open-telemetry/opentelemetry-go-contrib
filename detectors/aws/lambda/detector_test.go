// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// setLambdaEnv puts the process in a Lambda-like environment so the detector
// returns a populated resource rather than an empty one.
func setLambdaEnv(t *testing.T) {
	t.Helper()
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogGroupNameEnvVar, "/aws/lambda/testFunction")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")
}

// TestComposition_MergeWithDefault guards against schema URL drift between this
// detector and the SDK. [resource.Merge] reports [resource.ErrSchemaURLConflict]
// and drops the schema URL when the two disagree, so this fails as soon as the
// semconv version here and the one behind [resource.Default] diverge.
func TestComposition_MergeWithDefault(t *testing.T) {
	setLambdaEnv(t)

	detected, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

// TestComposition_WithCoreDetectors asserts this detector composes with the
// detectors built into the SDK.
func TestComposition_WithCoreDetectors(t *testing.T) {
	setLambdaEnv(t)

	res, err := resource.New(t.Context(),
		resource.WithDetectors(NewResourceDetector()),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}
