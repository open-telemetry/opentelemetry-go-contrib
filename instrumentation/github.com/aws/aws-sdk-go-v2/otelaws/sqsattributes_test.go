// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func TestSQSDeleteMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSDeleteMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSDeleteQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSGetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSListDeadLetterSourceQueuesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListDeadLetterSourceQueuesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSListQueueTagsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListQueueTagsInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSPurgeQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.PurgeQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSReceiveMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ReceiveMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSRemovePermissionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.RemovePermissionInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSSendMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageBatchInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSSendMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSSetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SetQueueAttributesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSTagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.TagQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}

func TestSQSUntagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.UntagQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.NetPeerName("test-queue-url"))
}
