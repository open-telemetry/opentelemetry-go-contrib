// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"errors"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// MetricRecorder records metrics of a completed AWS SDK operation. It is the
// metrics counterpart of AttributeBuilder: in and out are the input and output
// of the operation, err is the error it returned, if any.
type MetricRecorder func(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput, err error)

// NewDefaultMetricRecorder creates the instruments of the built-in service
// metrics with meter and returns the MetricRecorder recording them. It is the
// metrics counterpart of DefaultAttributeBuilder: it checks to see if there
// are service specific metrics available to record for the AWS service.
//
// Pass the returned MetricRecorder to WithMetricRecorder together with custom
// metric recorders to keep the built-in service metrics. It only records
// operations of the middlewares appended by AppendMiddlewares.
//
// Instruments that fail to be created are reported to the global error
// handler and replaced by no-ops.
func NewDefaultMetricRecorder(meter metric.Meter) MetricRecorder {
	services := map[string]MetricRecorder{
		dynamodb.ServiceID: NewDynamoDBMetricRecorder(meter),
	}

	return func(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput, err error) {
		serviceID := v2Middleware.GetServiceID(ctx)

		if fn, ok := services[serviceID]; ok {
			fn(ctx, in, out, err)
		}
	}
}

func errorTypeAttr(err error) attribute.KeyValue {
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		if code := apiErr.ErrorCode(); code != "" {
			return semconv.ErrorTypeKey.String(code)
		}
	}
	return semconv.ErrorType(err)
}
