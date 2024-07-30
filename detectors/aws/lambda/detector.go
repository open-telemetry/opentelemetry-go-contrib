// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lambda // import "go.opentelemetry.io/contrib/detectors/aws/lambda"

import (
	"context"
	"errors"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// For a complete list of reserved environment variables in Lambda, see:
// https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html
const (
	lambdaFunctionNameEnvVar    = "AWS_LAMBDA_FUNCTION_NAME" //nolint:gosec // False positive G101: Potential hardcoded credentials. The function name is added as attribute per semantic conventions.
	awsRegionEnvVar             = "AWS_REGION"
	lambdaFunctionVersionEnvVar = "AWS_LAMBDA_FUNCTION_VERSION"
	lambdaLogStreamNameEnvVar   = "AWS_LAMBDA_LOG_STREAM_NAME"
	lambdaMemoryLimitEnvVar     = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
)

var (
	empty          = resource.Empty()
	errNotOnLambda = errors.New("process is not on Lambda, cannot detect environment variables from Lambda")
)

// resource detector collects resource information from Lambda environment.
type resourceDetector struct{}

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS Lambda resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{}
}

// Detect collects resource attributes available when running on lambda.
func (detector *resourceDetector) Detect(context.Context) (*resource.Resource, error) {
	// Lambda resources come from ENV
	lambdaName := os.Getenv(lambdaFunctionNameEnvVar)
	if len(lambdaName) == 0 {
		return empty, errNotOnLambda
	}
	awsRegion := os.Getenv(awsRegionEnvVar)
	functionVersion := os.Getenv(lambdaFunctionVersionEnvVar)
	// The instance attributes corresponds to the log stream name for AWS lambda,
	// see the FaaS resource specification for more details.
	instance := os.Getenv(lambdaLogStreamNameEnvVar)

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudRegion(awsRegion),
		semconv.FaaSInstance(instance),
		semconv.FaaSName(lambdaName),
		semconv.FaaSVersion(functionVersion),
	}

	maxMemoryStr := os.Getenv(lambdaMemoryLimitEnvVar)
	maxMemory, err := strconv.Atoi(maxMemoryStr)
	if err == nil {
		attrs = append(attrs, semconv.FaaSMaxMemory(maxMemory))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
