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
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// SNSAttributeSetter sets SNS specific attributes depending on the SNS operation being performed.
func SNSAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	snsAttributes := []attribute.KeyValue{semconv.MessagingSystem("AmazonSNS")}

	switch v := in.Parameters.(type) {
	case *sns.PublishInput:
		var value string
		valuePointer := v.TargetArn
		if valuePointer == nil {
			valuePointer = v.TopicArn
		}
		if valuePointer == nil {
			if v.PhoneNumber != nil {
				value = *v.PhoneNumber
			}
		} else {
			arnParts := strings.Split(*valuePointer, ":")
			value = arnParts[len(arnParts)-1]
		}
		if value != "" {
			snsAttributes = append(snsAttributes, semconv.MessagingDestinationName(value))
		}
		snsAttributes = append(snsAttributes, semconv.MessagingDestinationKindTopic)
	case *sns.PublishBatchInput:
		if v.TopicArn != nil {
			arnParts := strings.Split(*v.TopicArn, ":")
			value := arnParts[len(arnParts)-1]
			snsAttributes = append(snsAttributes, semconv.MessagingDestinationName(value))
		}
		snsAttributes = append(snsAttributes, semconv.MessagingDestinationKindTopic)
	}

	return snsAttributes
}
