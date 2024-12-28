// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func TestPublishInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{
			TopicArn: aws.String("arn:aws:sns:us-east-2:444455556666:my-topic"),
		},
	}

	attributes := SNSAttributeBuilder(context.Background(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName("my-topic"))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypePublish)
}

func TestPublishInputWithNoDestination(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{},
	}

	attributes := SNSAttributeBuilder(context.Background(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(""))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypePublish)
}

func TestPublishBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishBatchInput{
			TopicArn:                   aws.String("arn:aws:sns:us-east-2:444455556666:my-topic-batch"),
			PublishBatchRequestEntries: []types.PublishBatchRequestEntry{},
		},
	}

	attributes := SNSAttributeBuilder(context.Background(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName("my-topic-batch"))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_batch_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypePublish)
	assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
}
