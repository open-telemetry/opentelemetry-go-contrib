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

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// SQSAttributeSetter sets SQS specific attributes depending on the SQS operation being performed.
func SQSAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	sqsAttributes := []attribute.KeyValue{semconv.MessagingSystemKey.String("AmazonSQS")}

	switch v := in.Parameters.(type) {
	case *sqs.DeleteMessageBatchInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.DeleteMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.DeleteQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.GetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.ListDeadLetterSourceQueuesInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.ListQueueTagsInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.PurgeQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.ReceiveMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.RemovePermissionInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.SendMessageBatchInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.SendMessageInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.SetQueueAttributesInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.TagQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	case *sqs.UntagQueueInput:
		sqsAttributes = append(sqsAttributes, semconv.MessagingURLKey.String(*v.QueueUrl))
	}

	return sqsAttributes
}
