// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// SNSAttributeSetter sets SNS specific attributes depending on the SNS operation is being performed.
//
// Deprecated: Use SNSAttributeBuilder instead. This will be removed in a future release.
func SNSAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	return SNSAttributeBuilder(ctx, in, middleware.InitializeOutput{})
}

// SNSAttributeBuilder sets SNS specific attributes depending on the SNS operation is being performed.
func SNSAttributeBuilder(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue {
	snsAttributes := []attribute.KeyValue{semconv.MessagingSystemKey.String("aws_sns")}

	switch v := in.Parameters.(type) {
	case *sns.PublishBatchInput:
		snsAttributes = append(snsAttributes,
			semconv.MessagingDestinationName(extractDestinationName(v.TopicArn, nil)),
			semconv.MessagingOperationTypePublish,
			semconv.MessagingOperationName("publish_batch_input"),
			semconv.MessagingBatchMessageCount(len(v.PublishBatchRequestEntries)),
		)
	case *sns.PublishInput:
		snsAttributes = append(snsAttributes,
			semconv.MessagingDestinationName(extractDestinationName(v.TopicArn, v.TargetArn)),
			semconv.MessagingOperationTypePublish,
			semconv.MessagingOperationName("publish_input"),
		)
	}

	return snsAttributes
}

func extractDestinationName(topicArn, targetArn *string) string {
	if topicArn != nil && *topicArn != "" {
		return (*topicArn)[strings.LastIndex(*topicArn, ":")+1:]
	} else if targetArn != nil && *targetArn != "" {
		return (*targetArn)[strings.LastIndex(*targetArn, ":")+1:]
	}
	return ""
}
