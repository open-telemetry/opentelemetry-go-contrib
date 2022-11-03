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

package otelaws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func TestSQSDeleteMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSDeleteMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSDeleteQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.DeleteQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSGetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSListDeadLetterSourceQueuesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListDeadLetterSourceQueuesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSListQueueTagsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ListQueueTagsInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSPurgeQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.PurgeQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSReceiveMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.ReceiveMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSRemovePermissionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.RemovePermissionInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSSendMessageBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageBatchInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSSendMessageInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SendMessageInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSSetQueueAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.SetQueueAttributesInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSTagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.TagQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}

func TestSQSUntagQueueInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sqs.UntagQueueInput{
			QueueUrl: aws.String("test-queue-url"),
		},
	}

	attributes := SQSAttributeSetter(context.TODO(), input)

	assert.Contains(t, attributes, semconv.MessagingURLKey.String("test-queue-url"))
}
