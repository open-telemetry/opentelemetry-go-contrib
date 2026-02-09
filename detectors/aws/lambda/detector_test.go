// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
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
		semconv.FaaSMaxMemory(128 * miB),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := resourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err, "Detector unexpectedly returned error")
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// return empty resource when not running on lambda.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := resourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.Equal(t, errNotOnLambda, err)
	assert.Empty(t, res.Attributes())
}

// successfully detect cloud.account.id when the symlink is present.
func TestDetectAccountIDSymlink(t *testing.T) {
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")

	original := readlinkFunc
	readlinkFunc = func(string) (string, error) {
		return "123456789012", nil
	}
	t.Cleanup(func() { readlinkFunc = original })

	detector := resourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	// Verify the account ID attribute is present among the resource attributes.
	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.CloudAccountIDKey && attr.Value.AsString() == "123456789012" {
			found = true
			break
		}
	}
	assert.True(t, found, "cloud.account.id attribute not found in resource")
}

// cloud.account.id is absent when the symlink does not exist.
func TestDetectAccountIDSymlinkMissing(t *testing.T) {
	t.Setenv(lambdaFunctionNameEnvVar, "testFunction")
	t.Setenv(awsRegionEnvVar, "us-texas-1")
	t.Setenv(lambdaFunctionVersionEnvVar, "$LATEST")
	t.Setenv(lambdaLogStreamNameEnvVar, "2023/01/01/[$LATEST]5d1edb9e525d486696cf01a3503487bc")
	t.Setenv(lambdaMemoryLimitEnvVar, "128")

	original := readlinkFunc
	readlinkFunc = func(string) (string, error) {
		return "", errors.New("no such file")
	}
	t.Cleanup(func() { readlinkFunc = original })

	detector := resourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	// Verify no cloud.account.id attribute is present.
	for _, attr := range res.Attributes() {
		assert.NotEqual(t, semconv.CloudAccountIDKey, attr.Key, "cloud.account.id should not be present when symlink is missing")
	}
}
