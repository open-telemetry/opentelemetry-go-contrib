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

package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithyauth "github.com/aws/smithy-go/auth"
	"github.com/aws/smithy-go/middleware"
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

type dynamoDBAuthResolver struct{}

func (r *dynamoDBAuthResolver) ResolveAuthSchemes(context.Context, *dynamodb.AuthResolverParameters) ([]*smithyauth.Option, error) {
	return []*smithyauth.Option{
		{SchemeID: smithyauth.SchemeIDAnonymous},
	}, nil
}

func TestDynamodbTags(t *testing.T) {
	cases := struct {
		responseStatus     int
		expectedRegion     string
		expectedStatusCode int
		expectedError      codes.Code
	}{
		responseStatus:     http.StatusOK,
		expectedRegion:     "us-west-2",
		expectedStatusCode: http.StatusOK,
	}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(cases.responseStatus)
		}))
	defer server.Close()

	t.Run("dynamodb tags", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		svc := dynamodb.New(dynamodb.Options{
			Region:             cases.expectedRegion,
			BaseEndpoint:       &server.URL,
			AuthSchemeResolver: &dynamoDBAuthResolver{},
			AuthSchemes: []smithyhttp.AuthScheme{
				smithyhttp.NewAnonymousScheme(),
			},
			Retryer: aws.NopRetryer{},
		})

		_, err := svc.GetItem(context.Background(), &dynamodb.GetItemInput{
			TableName:            aws.String("table1"),
			ConsistentRead:       aws.Bool(false),
			ProjectionExpression: aws.String("test"),
			Key: map[string]dtypes.AttributeValue{
				"id": &dtypes.AttributeValueMemberS{Value: "test"},
			},
		}, func(options *dynamodb.Options) {
			otelaws.AppendMiddlewares(
				&options.APIOptions, otelaws.WithAttributeSetter(otelaws.DynamoDBAttributeSetter), otelaws.WithTracerProvider(provider))
		})

		if cases.expectedError == codes.Unset {
			assert.NoError(t, err)
		} else {
			assert.NotNil(t, err)
		}

		spans := sr.Ended()
		require.Len(t, spans, 1)
		span := spans[0]

		assert.Equal(t, "DynamoDB.GetItem", span.Name())
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())
		attrs := span.Attributes()
		assert.Contains(t, attrs, attribute.Int("http.status_code", cases.expectedStatusCode))
		assert.Contains(t, attrs, attribute.String("rpc.service", "DynamoDB"))
		assert.Contains(t, attrs, attribute.String("aws.region", cases.expectedRegion))
		assert.Contains(t, attrs, attribute.String("rpc.method", "GetItem"))
		assert.Contains(t, attrs, attribute.String("rpc.system", "aws-api"))
		assert.Contains(t, attrs, attribute.StringSlice(
			"aws.dynamodb.table_names", []string{"table1"},
		))
		assert.Contains(t, attrs, attribute.String("aws.dynamodb.projection", "test"))
		assert.Contains(t, attrs, attribute.Bool("aws.dynamodb.consistent_read", false))
	})
}

func TestDynamodbTagsCustomSetter(t *testing.T) {
	cases := struct {
		responseStatus     int
		expectedRegion     string
		expectedStatusCode int
		expectedError      codes.Code
	}{
		responseStatus:     http.StatusOK,
		expectedRegion:     "us-west-2",
		expectedStatusCode: http.StatusOK,
	}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(cases.responseStatus)
		}))
	defer server.Close()

	t.Run("dynamodb tags", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

		svc := dynamodb.New(dynamodb.Options{
			Region:             cases.expectedRegion,
			BaseEndpoint:       &server.URL,
			AuthSchemeResolver: &dynamoDBAuthResolver{},
			Retryer:            aws.NopRetryer{},
		})

		mycustomsetter := otelaws.AttributeSetter(func(context.Context, middleware.InitializeInput) []attribute.KeyValue {
			customAttributes := []attribute.KeyValue{
				{
					Key:   "customattribute2key",
					Value: attribute.StringValue("customattribute2value"),
				},
				{
					Key:   "customattribute1key",
					Value: attribute.StringValue("customattribute1value"),
				},
			}

			return customAttributes
		})

		_, err := svc.GetItem(context.Background(), &dynamodb.GetItemInput{
			TableName:            aws.String("table1"),
			ConsistentRead:       aws.Bool(false),
			ProjectionExpression: aws.String("test"),
			Key: map[string]dtypes.AttributeValue{
				"id": &dtypes.AttributeValueMemberS{Value: "test"},
			},
		}, func(options *dynamodb.Options) {
			otelaws.AppendMiddlewares(
				&options.APIOptions, otelaws.WithAttributeSetter(otelaws.DynamoDBAttributeSetter, mycustomsetter), otelaws.WithTracerProvider(provider))
		})

		if cases.expectedError == codes.Unset {
			assert.NoError(t, err)
		} else {
			assert.NotNil(t, err)
		}

		spans := sr.Ended()
		require.Len(t, spans, 1)
		span := spans[0]

		assert.Equal(t, "DynamoDB.GetItem", span.Name())
		assert.Equal(t, trace.SpanKindClient, span.SpanKind())
		attrs := span.Attributes()
		assert.Contains(t, attrs, attribute.Int("http.status_code", cases.expectedStatusCode))
		assert.Contains(t, attrs, attribute.String("rpc.service", "DynamoDB"))
		assert.Contains(t, attrs, attribute.String("aws.region", cases.expectedRegion))
		assert.Contains(t, attrs, attribute.String("rpc.method", "GetItem"))
		assert.Contains(t, attrs, attribute.StringSlice(
			"aws.dynamodb.table_names", []string{"table1"},
		))
		assert.Contains(t, attrs, attribute.String("aws.dynamodb.projection", "test"))
		assert.Contains(t, attrs, attribute.Bool("aws.dynamodb.consistent_read", false))
		assert.Contains(t, attrs, attribute.String("customattribute2key", "customattribute2value"))
		assert.Contains(t, attrs, attribute.String("customattribute1key", "customattribute1value"))
	})
}
