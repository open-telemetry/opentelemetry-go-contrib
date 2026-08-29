// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package lambda provides a resource detector for AWS Lambda.
package lambda

import (
	"context"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// For a complete list of reserved environment variables in Lambda, see:
// https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html
const (
	lambdaFunctionNameEnvVar    = "AWS_LAMBDA_FUNCTION_NAME"
	awsRegionEnvVar             = "AWS_REGION"
	lambdaFunctionVersionEnvVar = "AWS_LAMBDA_FUNCTION_VERSION"
	lambdaLogGroupNameEnvVar    = "AWS_LAMBDA_LOG_GROUP_NAME"
	lambdaLogStreamNameEnvVar   = "AWS_LAMBDA_LOG_STREAM_NAME"
	lambdaMemoryLimitEnvVar     = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
	miB                         = 1 << 20
)

var empty = resource.Empty()

type config struct {
	filter attribute.Filter
}

// Option configures a resource detector.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// resource detector collects resource information from Lambda environment.
type resourceDetector struct {
	cfg config
}

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS Lambda resources.
func NewResourceDetector(opts ...Option) resource.Detector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &resourceDetector{cfg: cfg}
}

// Detect collects resource attributes available when running on lambda.
func (d *resourceDetector) Detect(context.Context) (*resource.Resource, error) {
	// Lambda resources come from ENV
	lambdaName := os.Getenv(lambdaFunctionNameEnvVar)
	if lambdaName == "" {
		return empty, nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSLambda,
		semconv.FaaSName(lambdaName),
	}

	if v, ok := os.LookupEnv(awsRegionEnvVar); ok {
		attrs = append(attrs, semconv.CloudRegion(v))
	}
	if v, ok := os.LookupEnv(lambdaFunctionVersionEnvVar); ok {
		attrs = append(attrs, semconv.FaaSVersion(v))
	}
	// The instance attribute corresponds to the log stream name for AWS lambda,
	// see the FaaS resource specification for more details.
	if v, ok := os.LookupEnv(lambdaLogStreamNameEnvVar); ok {
		attrs = append(attrs, semconv.FaaSInstance(v), semconv.AWSLogStreamNames(v))
	}
	if v, ok := os.LookupEnv(lambdaLogGroupNameEnvVar); ok {
		attrs = append(attrs, semconv.AWSLogGroupNames(v))
	}
	// faas.max_memory is measured in bytes, the environment variable reports MiB.
	if maxMemory, err := strconv.Atoi(os.Getenv(lambdaMemoryLimitEnvVar)); err == nil {
		attrs = append(attrs, semconv.FaaSMaxMemory(maxMemory*miB))
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
