// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

var (
	serverAddress = "sqs.us-east-1.amazonaws.com"
	queueName     = "some_queue_name"
	queueUrl      = fmt.Sprintf("https://%s/000000000000/%s", serverAddress, queueName)
)

func TestSQSDeleteMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageBatchInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSettle)
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSDeleteMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSettle)
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSDeleteQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteQueueInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSGetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.GetQueueAttributesInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSListDeadLetterSourceQueuesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListDeadLetterSourceQueuesInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSListQueueTagsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListQueueTagsInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSPurgeQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.PurgeQueueInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSReceiveMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ReceiveMessageInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeReceive)
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSRemovePermissionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.RemovePermissionInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSSendMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageBatchInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSSendMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageInput{
			MessageBody: aws.String(""),
			QueueUrl:    &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingMessageBodySize(0))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSSetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SetQueueAttributesInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSTagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.TagQueueInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}

func TestSQSUntagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.UntagQueueInput{
			QueueUrl: &queueUrl,
		},
	}

	attributes := SQSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSSQSQueueURL(queueUrl))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(queueName))
	assert.Contains(t, attributes, semconv.MessagingSystemAWSSQS)
	assert.Contains(t, attributes, semconv.ServerAddress(serverAddress))
}
