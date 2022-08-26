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

package lambda // import "go.opentelemetry.io/contrib/detectors/aws/lambda"

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// For a complete list of reserved environment variables in Lambda, see:
// https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html
const (
	lambdaFunctionNameEnvVar    = "AWS_LAMBDA_FUNCTION_NAME"
	awsRegionEnvVar             = "AWS_REGION"
	lambdaFunctionVersionEnvVar = "AWS_LAMBDA_FUNCTION_VERSION"
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

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudRegionKey.String(awsRegion),
		semconv.FaaSNameKey.String(lambdaName),
		semconv.FaaSVersionKey.String(functionVersion),
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
