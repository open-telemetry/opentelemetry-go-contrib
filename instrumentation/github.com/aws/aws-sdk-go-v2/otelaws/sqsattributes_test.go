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

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func TestSQSAttributeSetter(t *testing.T) {
	queueURL := "test-queue-url"
	awsQueueURL := aws.String(queueURL)
	inputs := map[string]middleware.InitializeInput{
		"with DeleteMessageBatchInput": {
			Parameters: &sqs.DeleteMessageBatchInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with DeleteMessageInput": {
			Parameters: &sqs.DeleteMessageInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with DeleteQueueInput": {
			Parameters: &sqs.DeleteQueueInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with GetQueueAttributesInput": {
			Parameters: &sqs.GetQueueAttributesInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with ListDeadLetterSourceQueuesInput": {
			Parameters: &sqs.ListDeadLetterSourceQueuesInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with ListQueueTagsInput": {
			Parameters: &sqs.ListQueueTagsInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with PurgeQueueInput": {
			Parameters: &sqs.PurgeQueueInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with ReceiveMessageInput": {
			Parameters: &sqs.ReceiveMessageInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with RemovePermissionInput": {
			Parameters: &sqs.RemovePermissionInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with SendMessageBatchInput": {
			Parameters: &sqs.SendMessageBatchInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with SendMessageInput": {
			Parameters: &sqs.SendMessageInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with SetQueueAttributesInput": {
			Parameters: &sqs.SetQueueAttributesInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with TagQueueInput": {
			Parameters: &sqs.TagQueueInput{
				QueueUrl: awsQueueURL,
			},
		},
		"with UntagQueueInput": {
			Parameters: &sqs.UntagQueueInput{
				QueueUrl: awsQueueURL,
			},
		},
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			attributes := SQSAttributeSetter(context.TODO(), input, &AttributeSettersConfig{})

			assert.Contains(t, attributes, semconv.NetPeerName(queueURL))
		})
	}
}
