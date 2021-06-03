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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
)

func TestAppendMiddlewares(t *testing.T) {
	cases := map[string]struct {
		responseStatus     int
		responseBody       []byte
		expectedRegion     string
		expectedError      string
		expectedRequestID  string
		expectedStatusCode int
	}{
		"invalidChangeBatchError": {
			responseStatus: 500,
			responseBody: []byte(`<?xml version="1.0" encoding="UTF-8"?>
		<InvalidChangeBatch xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
		  <Messages>
		    <Message>Tried to create resource record set duplicate.example.com. type A, but it already exists</Message>
		  </Messages>
		  <RequestId>b25f48e8-84fd-11e6-80d9-574e0c4664cb</RequestId>
		</InvalidChangeBatch>`),
			expectedRegion:     "us-east-1",
			expectedError:      "Error",
			expectedRequestID:  "b25f48e8-84fd-11e6-80d9-574e0c4664cb",
			expectedStatusCode: 500,
		},

		"standardRestXMLError": {
			responseStatus: 404,
			responseBody: []byte(`<?xml version="1.0"?>
		<ErrorResponse xmlns="http://route53.amazonaws.com/doc/2016-09-07/">
		  <Error>
		    <Type>Sender</Type>
		    <Code>MalformedXML</Code>
		    <Message>1 validation error detected: Value null at 'route53#ChangeSet' failed to satisfy constraint: Member must not be null</Message>
		  </Error>
		  <RequestId>1234567890A</RequestId>
		</ErrorResponse>
		`),
			expectedRegion:     "us-west-1",
			expectedError:      "Error",
			expectedRequestID:  "1234567890A",
			expectedStatusCode: 404,
		},

		"Success response": {
			responseStatus: 200,
			responseBody: []byte(`<?xml version="1.0" encoding="UTF-8"?>
		<ChangeResourceRecordSetsResponse>
   			<ChangeInfo>
      		<Comment>mockComment</Comment>
      		<Id>mockID</Id>
   		</ChangeInfo>
		</ChangeResourceRecordSetsResponse>`),
			expectedRegion:     "us-west-2",
			expectedStatusCode: 200,
		},
	}

	for name, c := range cases {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.responseStatus)
				_, err := w.Write(c.responseBody)
				if err != nil {
					t.Fatal(err)
				}
			}))
		defer server.Close()

		t.Run(name, func(t *testing.T) {
			sr := new(oteltest.SpanRecorder)
			provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

			svc := route53.NewFromConfig(aws.Config{
				Region: c.expectedRegion,
				EndpointResolver: aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:         server.URL,
						SigningName: "route53",
					}, nil
				}),
				Retryer: func() aws.Retryer {
					return aws.NopRetryer{}
				},
			})
			_, err := svc.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &types.ChangeBatch{
					Changes: []types.Change{},
					Comment: aws.String("mock"),
				},
				HostedZoneId: aws.String("zone"),
			}, func(options *route53.Options) {
				AppendMiddlewares(
					&options.APIOptions, WithTracerProvider(provider))
			})

			spans := sr.Completed()
			assert.Len(t, spans, 1)
			span := spans[0]

			if e, a := "Route 53", span.Name(); !strings.EqualFold(e, a) {
				t.Errorf("expected span name to be %s, got %s", e, a)
			}

			if e, a := trace.SpanKindClient, span.SpanKind(); e != a {
				t.Errorf("expected span kind to be %v, got %v", e, a)
			}

			if e, a := c.expectedError, span.StatusCode().String(); err != nil && !strings.EqualFold(e, a) {
				t.Errorf("Span Error is missing.")
			}

			if e, a := c.expectedStatusCode, span.Attributes()["http.status_code"].AsInt64(); e != int(a) {
				t.Errorf("expected status code to be %v, got %v", e, a)
			}

			if e, a := c.expectedRequestID, span.Attributes()["aws.request_id"].AsString(); !strings.EqualFold(e, a) {
				t.Errorf("expected request id to be %s, got %s", e, a)
			}

			if e, a := "Route 53", span.Attributes()["aws.service"].AsString(); !strings.EqualFold(e, a) {
				t.Errorf("expected service to be %s, got %s", e, a)
			}

			if e, a := c.expectedRegion, span.Attributes()["aws.region"].AsString(); !strings.EqualFold(e, a) {
				t.Errorf("expected region to be %s, got %s", e, a)
			}

			if e, a := "ChangeResourceRecordSets", span.Attributes()["aws.operation"].AsString(); !strings.EqualFold(e, a) {
				t.Errorf("expected operation to be %s, got %s", e, a)
			}
		})

	}
}
