// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestPublishInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{
			TopicArn: aws.String("arn:aws:sns:us-east-2:444455556666:my-topic"),
		},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemAWSSNS)
	assert.Contains(t, attributes, semconv.MessagingDestinationName("my-topic"))
	assert.Contains(t, attributes, semconv.AWSSNSTopicARN("arn:aws:sns:us-east-2:444455556666:my-topic"))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
}

func TestPublishInputWithNoDestination(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemAWSSNS)
	assert.Contains(t, attributes, semconv.MessagingDestinationName(""))
	for _, attr := range attributes {
		assert.NotEqual(t, semconv.AWSSNSTopicARNKey, attr.Key)
	}
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
}

func TestPublishInputWithTargetARN(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{
			TargetArn: aws.String("arn:aws:sns:us-east-2:444455556666:endpoint/GCM/app/device"),
		},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingDestinationName("endpoint/GCM/app/device"))
	for _, attr := range attributes {
		assert.NotEqual(t, semconv.AWSSNSTopicARNKey, attr.Key)
	}
}

func TestPublishBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishBatchInput{
			TopicArn:                   aws.String("arn:aws:sns:us-east-2:444455556666:my-topic-batch"),
			PublishBatchRequestEntries: []types.PublishBatchRequestEntry{},
		},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemAWSSNS)
	assert.Contains(t, attributes, semconv.MessagingDestinationName("my-topic-batch"))
	assert.Contains(t, attributes, semconv.AWSSNSTopicARN("arn:aws:sns:us-east-2:444455556666:my-topic-batch"))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_batch_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
	assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
}

func TestPublishBatchInputWithNoTopicARN(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishBatchInput{},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	for _, attr := range attributes {
		assert.NotEqual(t, semconv.AWSSNSTopicARNKey, attr.Key)
	}
}
