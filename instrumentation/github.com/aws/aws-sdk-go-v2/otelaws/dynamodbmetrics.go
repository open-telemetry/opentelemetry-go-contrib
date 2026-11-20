// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.43.0/dbconv"
)

// NewDynamoDBMetricRecorder creates the db.client.operation.duration
// instrument with meter and returns the MetricRecorder recording the duration
// of DynamoDB operations to it. It records nothing for operations of other
// services, and only records operations of the middlewares appended by
// AppendMiddlewares.
//
// If the instrument fails to be created, the error is reported to the global
// error handler and the returned MetricRecorder records to a no-op instrument.
func NewDynamoDBMetricRecorder(meter metric.Meter) MetricRecorder {
	duration, err := dbconv.NewClientOperationDuration(
		meter,
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context, _ middleware.InitializeInput, _ middleware.InitializeOutput, opErr error) {
		if v2Middleware.GetServiceID(ctx) != dynamodb.ServiceID {
			return
		}
		elapsed, ok := OperationDuration(ctx)
		if !ok {
			return
		}

		attrs := make([]attribute.KeyValue, 0, 2)
		if operation := v2Middleware.GetOperationName(ctx); operation != "" {
			attrs = append(attrs, duration.AttrOperationName(operation))
		}
		if opErr != nil {
			attrs = append(attrs, errorTypeAttr(opErr))
		}

		duration.Record(ctx, elapsed.Seconds(), dbconv.SystemNameAWSDynamoDB, attrs...)
	}
}
