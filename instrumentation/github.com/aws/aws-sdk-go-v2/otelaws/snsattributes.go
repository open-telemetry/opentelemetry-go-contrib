// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// A topic name is bounded and admits only a fixed ASCII set: AWS documents it
// as made up of only uppercase and lowercase ASCII letters, numbers,
// underscores and hyphens, between 1 and 256 characters long, with a FIFO
// topic ending in the reserved .fifo suffix.
const (
	maxTopicNameLen = 256
	fifoSuffix      = ".fifo"
)

// SNSAttributeBuilder sets SNS specific attributes depending on the SNS operation is being performed.
func SNSAttributeBuilder(_ context.Context, in middleware.InitializeInput, _ middleware.InitializeOutput) []attribute.KeyValue {
	snsAttributes := []attribute.KeyValue{semconv.MessagingSystemKey.String("aws_sns")}

	switch v := in.Parameters.(type) {
	case *sns.PublishBatchInput:
		destination := selectPublishDestination(v.TopicArn)
		snsAttributes = append(
			snsAttributes,
			semconv.MessagingDestinationName(publishDestinationName(destination)),
			semconv.MessagingOperationTypeSend,
			semconv.MessagingOperationName("publish_batch_input"),
			semconv.MessagingBatchMessageCount(len(v.PublishBatchRequestEntries)),
		)
		snsAttributes = appendTopicARN(snsAttributes, destination)
	case *sns.PublishInput:
		destination := selectPublishDestination(v.TopicArn, v.TargetArn)
		snsAttributes = append(
			snsAttributes,
			semconv.MessagingDestinationName(publishDestinationName(destination)),
			semconv.MessagingOperationTypeSend,
			semconv.MessagingOperationName("publish_input"),
		)
		snsAttributes = appendTopicARN(snsAttributes, destination)
	}

	return snsAttributes
}

// selectPublishDestination returns the destination a publish request is
// addressed to, which is the first candidate field holding a value.
//
// Publish names its destination in TopicArn, TargetArn or PhoneNumber, so
// TargetArn is a destination form rather than an endpoint-only field: the API
// reference says that a request without TargetArn must supply PhoneNumber or
// TopicArn instead. AWS's published API model stated the overlap outright,
// describing TargetArn as "Either TopicArn or EndpointArn, but not both", and
// that sentence shipped in the SNS model up to botocore 1.12.x before it was
// dropped. TargetArn therefore holds either a topic ARN or a platform
// endpoint ARN, and both ARN fields are offered here.
//
// Only one of them can be the destination, and TopicArn takes precedence, so
// selecting it once here is what keeps every attribute derived from it
// describing the same destination.
func selectPublishDestination(candidates ...*string) string {
	for _, candidate := range candidates {
		if candidate != nil && *candidate != "" {
			return *candidate
		}
	}

	return ""
}

// publishDestinationName returns the name section of an addressed destination,
// which for an ARN is the part after the last colon.
func publishDestinationName(destination string) string {
	return destination[strings.LastIndex(destination, ":")+1:]
}

// appendTopicARN appends the ARN of the topic a publish request is addressed
// to, which keeps topics sharing a name in different accounts or regions
// distinct.
//
// The attribute is recorded only when the selected destination really is a
// topic ARN, which leaves it unset for a platform endpoint, a platform
// application, a subscription, and a value that is not a well formed SNS topic
// ARN at all. A request addressed by PhoneNumber names no ARN and so arrives
// here with an empty destination.
func appendTopicARN(attrs []attribute.KeyValue, destination string) []attribute.KeyValue {
	if !isTopicARN(destination) {
		return attrs
	}

	return append(attrs, semconv.AWSSNSTopicARN(destination))
}

// isTopicARN reports whether value is the ARN of an SNS topic.
//
// A topic ARN holds the bare topic name in the resource section, as in
// arn:aws:sns:us-east-2:123456789012:my-topic, and is always partitioned,
// regional and account owned.
func isTopicARN(value string) bool {
	parsed, err := arn.Parse(value)
	if err != nil {
		return false
	}

	if parsed.Service != "sns" ||
		parsed.Partition == "" ||
		parsed.Region == "" ||
		parsed.AccountID == "" {
		return false
	}

	return isTopicName(parsed.Resource)
}

// isTopicName reports whether resource is a bare SNS topic name.
//
// Requiring the name rather than only rejecting separators is what tells a
// topic apart from the other SNS resources. arn.Parse splits off just the
// first five ARN sections and leaves everything after them in Resource, so a
// subscription keeps its subscription ID there after a colon, while a platform
// application or platform endpoint uses a slash separated path. Neither
// character is in the topic name charset, so both are rejected here along with
// everything else outside it: spaces, control characters, non-ASCII bytes, the
// leading plus of a phone number, and a name over the length bound.
func isTopicName(resource string) bool {
	if resource == "" || len(resource) > maxTopicNameLen {
		return false
	}

	// A FIFO topic carries the reserved suffix inside that same bound, and the
	// suffix is the one place a dot is allowed.
	name := strings.TrimSuffix(resource, fifoSuffix)
	if name == "" {
		return false
	}

	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-',
			c == '_':
		default:
			return false
		}
	}

	return true
}
