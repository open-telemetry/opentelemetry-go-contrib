// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// SQSAttributeBuilder sets SQS specific attributes depending on the SQS operation being performed.
func SQSAttributeBuilder(_ context.Context, in middleware.InitializeInput, _ middleware.InitializeOutput) []attribute.KeyValue {
	sqsAttributes := []attribute.KeyValue{semconv.MessagingSystemAWSSQS}

	switch v := in.Parameters.(type) {
	case *sqs.DeleteMessageBatchInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl), semconv.MessagingOperationTypeSettle, semconv.MessagingBatchMessageCount(len(v.Entries)))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.DeleteMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl), semconv.MessagingOperationTypeSettle)
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.DeleteQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.GetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.ListDeadLetterSourceQueuesInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.ListQueueTagsInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.PurgeQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.ReceiveMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl), semconv.MessagingOperationTypeReceive)
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.RemovePermissionInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.SendMessageBatchInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl), semconv.MessagingOperationTypeSend, semconv.MessagingBatchMessageCount(len(v.Entries)))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.SendMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl), semconv.MessagingOperationTypeSend, semconv.MessagingMessageBodySize(len(*v.MessageBody)))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.SetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.TagQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	case *sqs.UntagQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.AWSSQSQueueURL(*v.QueueUrl))
		sqsAttributes = append(sqsAttributes, queueUrlAttrs(*v.QueueUrl)...)
	}

	return sqsAttributes
}

func queueUrlAttrs(queueUrl string) []attribute.KeyValue {
	var attrs []attribute.KeyValue

	parts, err := url.Parse(queueUrl)
	if err != nil {
		return nil
	}

	if addr, port, err := net.SplitHostPort(parts.Host); err == nil {
		if port, err := strconv.Atoi(port); err == nil {
			attrs = append(attrs, semconv.ServerAddress(addr), semconv.ServerPort(port))
		}
	} else {
		attrs = append(attrs, semconv.ServerAddress(parts.Host))
	}

	if _, queuename, found := strings.Cut(strings.TrimPrefix(parts.Path, "/"), "/"); found {
		attrs = append(attrs, semconv.MessagingDestinationName(queuename))
	}

	return attrs
}
