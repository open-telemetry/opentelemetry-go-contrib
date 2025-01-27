// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// SQSAttributeSetter sets SQS specific attributes depending on the SQS operation being performed.
//
// Deprecated: Use SQSAttributeBuilder instead. This will be removed in a future release.
func SQSAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	return SQSAttributeBuilder(ctx, in, middleware.InitializeOutput{})
}

// SQSAttributeBuilder sets SQS specific attributes depending on the SQS operation being performed.
func SQSAttributeBuilder(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue {
	sqsAttributes := []attribute.KeyValue{semconv.MessagingSystem("AmazonSQS")}

	key := semconv.NetPeerNameKey
	switch v := in.Parameters.(type) {
	case *sqs.DeleteMessageBatchInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.DeleteMessageInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.DeleteQueueInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.GetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.ListDeadLetterSourceQueuesInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.ListQueueTagsInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.PurgeQueueInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.ReceiveMessageInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.RemovePermissionInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.SendMessageBatchInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.SendMessageInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.SetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.TagQueueInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	case *sqs.UntagQueueInput:
		sqsAttributes = append(sqsAttributes, key.String(*v.QueueUrl))
	}

	return sqsAttributes
}
