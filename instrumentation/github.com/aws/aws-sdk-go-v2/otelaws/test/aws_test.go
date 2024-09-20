// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	smithyauth "github.com/aws/smithy-go/auth"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type route53AuthResolver struct{}

func (r *route53AuthResolver) ResolveAuthSchemes(context.Context, *route53.AuthResolverParameters) ([]*smithyauth.Option, error) {
	return []*smithyauth.Option{
		{SchemeID: smithyauth.SchemeIDAnonymous},
	}, nil
}

func TestAppendMiddlewares(t *testing.T) {
	cases := map[string]struct {
		responseStatus     int
		responseBody       []byte
		expectedRegion     string
		expectedError      codes.Code
		expectedRequestID  string
		expectedStatusCode int
	}{
		"invalidChangeBatchError": {
			responseStatus: http.StatusInternalServerError,
			responseBody: []byte(`<?xml version="1.0" encoding="UTF-8"?>
		<InvalidChangeBatch xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
		  <Messages>
		    <Message>Tried to create resource record set duplicate.example.com. type A, but it already exists</Message>
		  </Messages>
		  <RequestId>b25f48e8-84fd-11e6-80d9-574e0c4664cb</RequestId>
		</InvalidChangeBatch>`),
			expectedRegion:     "us-east-1",
			expectedError:      codes.Error,
			expectedRequestID:  "b25f48e8-84fd-11e6-80d9-574e0c4664cb",
			expectedStatusCode: http.StatusInternalServerError,
		},

		"standardRestXMLError": {
			responseStatus: http.StatusNotFound,
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
			expectedError:      codes.Error,
			expectedRequestID:  "1234567890A",
			expectedStatusCode: http.StatusNotFound,
		},

		"Success response": {
			responseStatus: http.StatusOK,
			responseBody: []byte(`<?xml version="1.0" encoding="UTF-8"?>
		<ChangeResourceRecordSetsResponse>
   			<ChangeInfo>
      		<Comment>mockComment</Comment>
      		<Id>mockID</Id>
   		</ChangeInfo>
		</ChangeResourceRecordSetsResponse>`),
			expectedRegion:     "us-west-2",
			expectedStatusCode: http.StatusOK,
		},
	}

	for name, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.responseStatus)
				_, err := w.Write(c.responseBody)
				if err != nil {
					t.Fatal(err)
				}
			}))

		t.Run(name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

			svc := route53.New(route53.Options{
				Region:             c.expectedRegion,
				BaseEndpoint:       &srv.URL,
				AuthSchemeResolver: &route53AuthResolver{},
				AuthSchemes: []smithyhttp.AuthScheme{
					smithyhttp.NewAnonymousScheme(),
				},
				Retryer: aws.NopRetryer{},
			})

			_, err := svc.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &types.ChangeBatch{
					Changes: []types.Change{},
					Comment: aws.String("mock"),
				},
				HostedZoneId: aws.String("zone"),
			}, func(options *route53.Options) {
				otelaws.AppendMiddlewares(
					&options.APIOptions, otelaws.WithTracerProvider(provider))
			})
			if c.expectedError == codes.Unset {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1)
			span := spans[0]

			assert.Equal(t, "Route 53.ChangeResourceRecordSets", span.Name())
			assert.Equal(t, trace.SpanKindClient, span.SpanKind())
			assert.Equal(t, c.expectedError, span.Status().Code)
			attrs := span.Attributes()
			assert.Contains(t, attrs, attribute.Int("http.status_code", c.expectedStatusCode))
			if c.expectedRequestID != "" {
				assert.Contains(t, attrs, attribute.String("aws.request_id", c.expectedRequestID))
			}
			assert.Contains(t, attrs, attribute.String("rpc.system", "aws-api"))
			assert.Contains(t, attrs, attribute.String("rpc.service", "Route 53"))
			assert.Contains(t, attrs, attribute.String("aws.region", c.expectedRegion))
			assert.Contains(t, attrs, attribute.String("rpc.method", "ChangeResourceRecordSets"))
		})

		srv.Close()
	}
}
