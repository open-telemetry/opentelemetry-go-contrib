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
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func TestSNSAttributeSetter(t *testing.T) {
	cases := map[string]struct {
		input     middleware.InitializeInput
		expect    []attribute.KeyValue
		notExpect []attribute.KeyValue
		context   context.Context
	}{
		"when publish input with target arn is passed": {
			middleware.InitializeInput{
				Parameters: &sns.PublishInput{
					TargetArn: aws.String("arn:aws:sns:us-east-1:0000000000:test-target-arn"),
				},
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationName("test-target-arn"),
				semconv.MessagingDestinationKindTopic,
			},
			nil,
			context.TODO(),
		},
		"when publish input with topic arn is passed": {
			middleware.InitializeInput{
				Parameters: &sns.PublishInput{
					TopicArn: aws.String("arn:aws:sns:us-east-1:0000000000:test-topic-arn"),
				},
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationName("test-topic-arn"),
				semconv.MessagingDestinationKindTopic,
			},
			nil,
			context.TODO(),
		},
		"when publish input with a phone number is passed and sensitive attributes are not recorded": {
			middleware.InitializeInput{
				Parameters: &sns.PublishInput{
					PhoneNumber: aws.String("+4900000000000"),
				},
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationKindTopic,
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationName("+4900000000000"),
			},
			context.TODO(),
		},
		"when publish input with a phone number is passed and sensitive attributes are recorded": {
			middleware.InitializeInput{
				Parameters: &sns.PublishInput{
					PhoneNumber: aws.String("+4900000000000"),
				},
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationName("+4900000000000"),
				semconv.MessagingDestinationKindTopic,
			},
			nil,
			injectConfig(context.TODO(), &config{RecordSNSPhoneNumber: true}),
		},
		"when publish batch input is passed": {
			middleware.InitializeInput{
				Parameters: &sns.PublishBatchInput{
					TopicArn: aws.String("arn:aws:sns:us-east-1:0000000000:test-topic-arn"),
				},
			},
			[]attribute.KeyValue{
				semconv.MessagingDestinationName("test-topic-arn"),
				semconv.MessagingDestinationKindTopic,
			},
			nil,
			context.TODO(),
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			attributes := SNSAttributeSetter(test.context, test.input)

			for _, expectation := range test.expect {
				assert.Contains(t, attributes, expectation)
			}

			for _, expectation := range test.notExpect {
				assert.NotContains(t, attributes, expectation)
			}
		})
	}
}
