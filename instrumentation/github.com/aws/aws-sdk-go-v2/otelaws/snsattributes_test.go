// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	// snsExampleARNPrefix is the leading ARN sections the examples below share,
	// so each one states only the resource section it is about.
	snsExampleARNPrefix = "arn:aws:sns:us-east-2:444455556666:"

	snsTopicName = "my-topic"
	snsTopicARN  = snsExampleARNPrefix + snsTopicName

	snsFIFOTopicName = snsTopicName + fifoSuffix
	snsFIFOTopicARN  = snsExampleARNPrefix + snsFIFOTopicName

	// snsTopicARNInOtherPartition and snsTopicARNInOtherAccount hold the same
	// topic name as snsTopicARN, which is what the attribute tells apart.
	snsTopicARNInOtherPartition = "arn:aws-cn:sns:cn-north-1:444455556666:" + snsTopicName
	snsTopicARNInOtherAccount   = "arn:aws:sns:us-east-2:111122223333:" + snsTopicName

	snsBatchTopicName = "my-topic-batch"
	snsBatchTopicARN  = "arn:aws:sns:eu-west-1:444455556666:" + snsBatchTopicName

	snsSubscriptionID  = "8a21d249-4329-4871-acc6-7be709c6ea7f"
	snsSubscriptionARN = snsTopicARN + ":" + snsSubscriptionID

	snsPlatformEndpointResource = "endpoint/APNS/my-app/00000000-1111-2222-3333-444455556666"
	snsPlatformEndpointARN      = snsExampleARNPrefix + snsPlatformEndpointResource

	snsPlatformApplicationResource = "app/APNS/my-app"
	snsPlatformApplicationARN      = snsExampleARNPrefix + snsPlatformApplicationResource

	snsExamplePhoneDigits = "5555550123"
	snsExamplePhoneNumber = "+1" + snsExamplePhoneDigits
)

func TestPublishInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{
			TopicArn: aws.String(snsTopicARN),
		},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(snsTopicName))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
}

func TestPublishInputWithNoDestination(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishInput{},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(""))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
	assertSNSTopicARN(t, attributes, "")
}

func TestPublishBatchInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &sns.PublishBatchInput{
			TopicArn:                   aws.String(snsBatchTopicARN),
			PublishBatchRequestEntries: []types.PublishBatchRequestEntry{},
		},
	}

	attributes := SNSAttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.MessagingSystemKey.String("aws_sns"))
	assert.Contains(t, attributes, semconv.MessagingDestinationName(snsBatchTopicName))
	assert.Contains(t, attributes, semconv.MessagingOperationName("publish_batch_input"))
	assert.Contains(t, attributes, semconv.MessagingOperationTypeSend)
	assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
}

// Publish names its destination with TopicArn, TargetArn or PhoneNumber. AWS's
// published API model described TargetArn as "Either TopicArn or EndpointArn,
// but not both" before that sentence was dropped, so a TargetArn holding a
// topic must be reported as the topic ARN and a TargetArn holding anything
// else must not be.
func TestPublishInputTopicARNAttribute(t *testing.T) {
	tests := []struct {
		name string
		in   sns.PublishInput
		// wantTopicARN empty means aws.sns.topic.arn must be absent.
		wantTopicARN    string
		wantDestination string
	}{
		{
			name:            "topic arn",
			in:              sns.PublishInput{TopicArn: aws.String(snsTopicARN)},
			wantTopicARN:    snsTopicARN,
			wantDestination: snsTopicName,
		},
		{
			name:            "fifo topic arn",
			in:              sns.PublishInput{TopicArn: aws.String(snsFIFOTopicARN)},
			wantTopicARN:    snsFIFOTopicARN,
			wantDestination: snsFIFOTopicName,
		},
		{
			name:            "topic arn in another partition",
			in:              sns.PublishInput{TopicArn: aws.String(snsTopicARNInOtherPartition)},
			wantTopicARN:    snsTopicARNInOtherPartition,
			wantDestination: snsTopicName,
		},
		{
			name:            "target arn naming a topic",
			in:              sns.PublishInput{TargetArn: aws.String(snsTopicARN)},
			wantTopicARN:    snsTopicARN,
			wantDestination: snsTopicName,
		},
		{
			name:            "target arn naming a fifo topic",
			in:              sns.PublishInput{TargetArn: aws.String(snsFIFOTopicARN)},
			wantTopicARN:    snsFIFOTopicARN,
			wantDestination: snsFIFOTopicName,
		},
		{
			name:            "target arn naming a platform endpoint",
			in:              sns.PublishInput{TargetArn: aws.String(snsPlatformEndpointARN)},
			wantDestination: snsPlatformEndpointResource,
		},
		{
			name:            "target arn naming a platform application",
			in:              sns.PublishInput{TargetArn: aws.String(snsPlatformApplicationARN)},
			wantDestination: snsPlatformApplicationResource,
		},
		{
			name:            "target arn naming a subscription",
			in:              sns.PublishInput{TargetArn: aws.String(snsSubscriptionARN)},
			wantDestination: snsSubscriptionID,
		},
		{
			name:            "target arn that is not an arn",
			in:              sns.PublishInput{TargetArn: aws.String(snsTopicName)},
			wantDestination: snsTopicName,
		},
		{
			name:            "target arn with too few sections",
			in:              sns.PublishInput{TargetArn: aws.String("arn:aws:sns:us-east-2:444455556666")},
			wantDestination: "444455556666",
		},
		{
			name:            "arn of another service",
			in:              sns.PublishInput{TargetArn: aws.String("arn:aws:sqs:us-east-2:444455556666:my-queue")},
			wantDestination: "my-queue",
		},
		{
			name:            "topic arn without a region",
			in:              sns.PublishInput{TopicArn: aws.String("arn:aws:sns::444455556666:" + snsTopicName)},
			wantDestination: snsTopicName,
		},
		{
			name:            "topic arn without an account",
			in:              sns.PublishInput{TopicArn: aws.String("arn:aws:sns:us-east-2::" + snsTopicName)},
			wantDestination: snsTopicName,
		},
		{
			name:            "topic arn with an empty resource",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix)},
			wantDestination: "",
		},
		{
			name:            "empty topic arn",
			in:              sns.PublishInput{TopicArn: aws.String("")},
			wantDestination: "",
		},
		{
			name:            "phone number",
			in:              sns.PublishInput{PhoneNumber: aws.String(snsExamplePhoneNumber)},
			wantDestination: "",
		},

		// A topic name admits only ASCII letters, numbers, underscores and
		// hyphens, within a bounded length, so a resource section outside that
		// set is some other thing and is not reported as a topic.
		{
			name:            "resource holding a single character name",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "a")},
			wantTopicARN:    snsExampleARNPrefix + "a",
			wantDestination: "a",
		},
		{
			name:            "resource holding underscores and hyphens",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "My_topic-1")},
			wantTopicARN:    snsExampleARNPrefix + "My_topic-1",
			wantDestination: "My_topic-1",
		},
		{
			name:            "resource holding a space",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "my topic")},
			wantDestination: "my topic",
		},
		{
			name:            "resource holding a control character",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "my\ntopic")},
			wantDestination: "my\ntopic",
		},
		{
			name:            "resource holding a non-ascii letter",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "my-t\u00f8pic")},
			wantDestination: "my-t\u00f8pic",
		},
		{
			name:            "resource holding a dot outside the fifo suffix",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + "my.topic")},
			wantDestination: "my.topic",
		},
		{
			name:            "resource holding a doubled fifo suffix",
			in:              sns.PublishInput{TopicArn: aws.String(snsFIFOTopicARN + fifoSuffix)},
			wantDestination: snsFIFOTopicName + fifoSuffix,
		},
		{
			name:            "resource that is only the fifo suffix",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + fifoSuffix)},
			wantDestination: fifoSuffix,
		},
		{
			name:            "resource that is a phone number",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + snsExamplePhoneNumber)},
			wantDestination: snsExamplePhoneNumber,
		},
		{
			name:            "resource at the length bound",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen))},
			wantTopicARN:    snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen),
			wantDestination: strings.Repeat("a", maxTopicNameLen),
		},
		{
			name:            "resource over the length bound",
			in:              sns.PublishInput{TopicArn: aws.String(snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen+1))},
			wantDestination: strings.Repeat("a", maxTopicNameLen+1),
		},
		{
			name: "fifo resource at the length bound",
			in: sns.PublishInput{
				TopicArn: aws.String(snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen-len(fifoSuffix)) + fifoSuffix),
			},
			wantTopicARN:    snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen-len(fifoSuffix)) + fifoSuffix,
			wantDestination: strings.Repeat("a", maxTopicNameLen-len(fifoSuffix)) + fifoSuffix,
		},
		{
			name: "fifo resource over the length bound",
			in: sns.PublishInput{
				TopicArn: aws.String(snsExampleARNPrefix + strings.Repeat("a", maxTopicNameLen-len(fifoSuffix)+1) + fifoSuffix),
			},
			wantDestination: strings.Repeat("a", maxTopicNameLen-len(fifoSuffix)+1) + fifoSuffix,
		},

		// A request carrying both fields names one destination, and TopicArn is
		// it whenever it holds a value. The topic ARN must come from that same
		// field, never from the field that was not selected.
		{
			name: "topic arn selected over a target arn naming a platform endpoint",
			in: sns.PublishInput{
				TopicArn:  aws.String(snsTopicARN),
				TargetArn: aws.String(snsPlatformEndpointARN),
			},
			wantTopicARN:    snsTopicARN,
			wantDestination: snsTopicName,
		},
		{
			name: "topic arn selected over a target arn naming another topic",
			in: sns.PublishInput{
				TopicArn:  aws.String(snsTopicARN),
				TargetArn: aws.String(snsTopicARNInOtherAccount),
			},
			wantTopicARN:    snsTopicARN,
			wantDestination: snsTopicName,
		},
		{
			name: "subscription topic arn does not fall through to a target arn topic",
			in: sns.PublishInput{
				TopicArn:  aws.String(snsSubscriptionARN),
				TargetArn: aws.String(snsTopicARN),
			},
			wantDestination: snsSubscriptionID,
		},
		{
			name: "platform endpoint topic arn does not fall through to a target arn topic",
			in: sns.PublishInput{
				TopicArn:  aws.String(snsPlatformEndpointARN),
				TargetArn: aws.String(snsTopicARN),
			},
			wantDestination: snsPlatformEndpointResource,
		},
		{
			name: "malformed topic arn does not fall through to a target arn topic",
			in: sns.PublishInput{
				TopicArn:  aws.String("not-an-arn"),
				TargetArn: aws.String(snsTopicARN),
			},
			wantDestination: "not-an-arn",
		},
		{
			name: "empty topic arn falls through to a target arn topic",
			in: sns.PublishInput{
				TopicArn:  aws.String(""),
				TargetArn: aws.String(snsTopicARN),
			},
			wantTopicARN:    snsTopicARN,
			wantDestination: snsTopicName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			attributes := SNSAttributeBuilder(
				t.Context(),
				middleware.InitializeInput{Parameters: &in},
				middleware.InitializeOutput{},
			)

			assert.Contains(t, attributes, semconv.MessagingDestinationName(tt.wantDestination))
			assertSNSTopicARN(t, attributes, tt.wantTopicARN)
		})
	}
}

// messaging.destination.name and aws.sns.topic.arn must describe one
// destination, so for every combination of the two ARN fields the recorded
// topic ARN must be the field that was selected, and its name section must be
// the recorded destination name. The precedence is restated here rather than
// taken from the code under test.
func TestPublishInputBothARNFieldsDescribeOneDestination(t *testing.T) {
	values := []struct {
		label string
		value string
	}{
		{"empty", ""},
		{"topic", snsTopicARN},
		{"fifo topic", snsFIFOTopicARN},
		{"topic in another account", snsTopicARNInOtherAccount},
		{"subscription", snsSubscriptionARN},
		{"platform endpoint", snsPlatformEndpointARN},
		{"platform application", snsPlatformApplicationARN},
		{"not an arn", "not-an-arn"},
	}

	for _, topic := range values {
		for _, target := range values {
			t.Run(topic.label+" with "+target.label+" target", func(t *testing.T) {
				attributes := SNSAttributeBuilder(t.Context(), middleware.InitializeInput{
					Parameters: &sns.PublishInput{
						TopicArn:  aws.String(topic.value),
						TargetArn: aws.String(target.value),
					},
				}, middleware.InitializeOutput{})

				// TopicArn is the destination whenever it holds a value.
				selected := topic.value
				if selected == "" {
					selected = target.value
				}
				wantName := selected[strings.LastIndex(selected, ":")+1:]

				assert.Contains(t, attributes, semconv.MessagingDestinationName(wantName))

				recorded := snsTopicARNValues(attributes)
				assert.LessOrEqual(t, len(recorded), 1, "aws.sns.topic.arn must be set at most once")
				for _, got := range recorded {
					assert.Equal(t, selected, got,
						"aws.sns.topic.arn must be the selected destination, not the other ARN field")
					assert.Equal(t, wantName, got[strings.LastIndex(got, ":")+1:],
						"aws.sns.topic.arn and messaging.destination.name must agree")
				}
			})
		}
	}
}

func TestPublishBatchInputTopicARNAttribute(t *testing.T) {
	tests := []struct {
		name string
		in   sns.PublishBatchInput
		// wantTopicARN empty means aws.sns.topic.arn must be absent.
		wantTopicARN    string
		wantDestination string
	}{
		{
			name:            "topic arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsBatchTopicARN)},
			wantTopicARN:    snsBatchTopicARN,
			wantDestination: snsBatchTopicName,
		},
		{
			name:            "fifo topic arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsFIFOTopicARN)},
			wantTopicARN:    snsFIFOTopicARN,
			wantDestination: snsFIFOTopicName,
		},
		{
			name:            "subscription arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsSubscriptionARN)},
			wantDestination: snsSubscriptionID,
		},
		{
			name:            "platform endpoint arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsPlatformEndpointARN)},
			wantDestination: snsPlatformEndpointResource,
		},
		{
			name:            "resource holding a space",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsExampleARNPrefix + "my topic")},
			wantDestination: "my topic",
		},
		{
			name:            "value that is not an arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String(snsBatchTopicName)},
			wantDestination: snsBatchTopicName,
		},
		{
			name:            "empty topic arn",
			in:              sns.PublishBatchInput{TopicArn: aws.String("")},
			wantDestination: "",
		},
		{
			name:            "no topic arn",
			in:              sns.PublishBatchInput{},
			wantDestination: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			in.PublishBatchRequestEntries = []types.PublishBatchRequestEntry{}

			attributes := SNSAttributeBuilder(
				t.Context(),
				middleware.InitializeInput{Parameters: &in},
				middleware.InitializeOutput{},
			)

			assert.Contains(t, attributes, semconv.MessagingDestinationName(tt.wantDestination))
			assert.Contains(t, attributes, semconv.MessagingBatchMessageCount(0))
			assertSNSTopicARN(t, attributes, tt.wantTopicARN)
		})
	}
}

// Two topics can share a name in one region across accounts. The topic ARN is
// what tells them apart, which is the reason to record it.
func TestPublishInputTopicARNDistinguishesSameTopicName(t *testing.T) {
	first := SNSAttributeBuilder(t.Context(), middleware.InitializeInput{
		Parameters: &sns.PublishInput{TopicArn: aws.String(snsTopicARN)},
	}, middleware.InitializeOutput{})

	second := SNSAttributeBuilder(t.Context(), middleware.InitializeInput{
		Parameters: &sns.PublishInput{TopicArn: aws.String(snsTopicARNInOtherAccount)},
	}, middleware.InitializeOutput{})

	assert.Contains(t, first, semconv.MessagingDestinationName(snsTopicName))
	assert.Contains(t, second, semconv.MessagingDestinationName(snsTopicName))
	assert.Contains(t, first, semconv.AWSSNSTopicARN(snsTopicARN))
	assert.Contains(t, second, semconv.AWSSNSTopicARN(snsTopicARNInOtherAccount))
}

// An SMS recipient is personal data, so no attribute may carry it.
func TestPublishInputPhoneNumberIsNeverRecorded(t *testing.T) {
	attributes := SNSAttributeBuilder(t.Context(), middleware.InitializeInput{
		Parameters: &sns.PublishInput{PhoneNumber: aws.String(snsExamplePhoneNumber)},
	}, middleware.InitializeOutput{})

	for _, attr := range attributes {
		assert.NotContains(t, attr.Value.String(), snsExamplePhoneDigits)
	}
}

// An operation that is not a publish carries no topic ARN.
func TestUnhandledInputHasNoTopicARN(t *testing.T) {
	attributes := SNSAttributeBuilder(t.Context(), middleware.InitializeInput{
		Parameters: &sns.CreateTopicInput{Name: aws.String(snsTopicName)},
	}, middleware.InitializeOutput{})

	assertSNSTopicARN(t, attributes, "")
}

// snsTopicARNValues returns every value recorded under aws.sns.topic.arn.
func snsTopicARNValues(attributes []attribute.KeyValue) []string {
	var values []string
	for _, attr := range attributes {
		if attr.Key == semconv.AWSSNSTopicARNKey {
			values = append(values, attr.Value.String())
		}
	}

	return values
}

// assertSNSTopicARN asserts that aws.sns.topic.arn holds want, or is absent
// when want is empty.
func assertSNSTopicARN(t *testing.T, attributes []attribute.KeyValue, want string) {
	t.Helper()

	got := snsTopicARNValues(attributes)
	if want == "" {
		assert.Empty(t, got, "aws.sns.topic.arn must not be set")
		return
	}

	assert.Equal(t, []string{want}, got)
}
