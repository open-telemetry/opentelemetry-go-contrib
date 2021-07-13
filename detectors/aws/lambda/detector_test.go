package lambda

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// successfully return resource when process is running on Amazon Lambda environment
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

// return empty resource when not running on lambda
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := resourceDetector{}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errNotOnLambda, err)
	assert.Equal(t, 0, len(res.Attributes()))
}