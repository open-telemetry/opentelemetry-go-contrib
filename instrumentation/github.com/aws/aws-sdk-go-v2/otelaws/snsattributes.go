// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// SNSAttributeBuilder sets SNS specific attributes depending on the SNS operation is being performed.
func SNSAttributeBuilder(_ context.Context, in middleware.InitializeInput, _ middleware.InitializeOutput) []attribute.KeyValue {
	snsAttributes := []attribute.KeyValue{semconv.MessagingSystemAWSSNS}

	switch v := in.Parameters.(type) {
	case *sns.PublishBatchInput:
		snsAttributes = append(snsAttributes,
			semconv.MessagingDestinationName(extractDestinationName(v.TopicArn, nil)),
			semconv.MessagingOperationTypeSend,
			semconv.MessagingOperationName("publish_batch_input"),
			semconv.MessagingBatchMessageCount(len(v.PublishBatchRequestEntries)),
		)
	case *sns.PublishInput:
		snsAttributes = append(snsAttributes,
			semconv.MessagingDestinationName(extractDestinationName(v.TopicArn, v.TargetArn)),
			semconv.MessagingOperationTypeSend,
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
