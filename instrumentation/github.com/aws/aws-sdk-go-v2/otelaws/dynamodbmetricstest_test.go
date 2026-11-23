// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

func TestDynamoDBClientOperationDuration(t *testing.T) {
	cases := []struct {
		name           string
		responseStatus int
		responseBody   string
		wantErr        bool
		wantErrorType  string
	}{
		{
			name:           "success",
			responseStatus: http.StatusOK,
			responseBody:   `{}`,
		},
		{
			name:           "error",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"__type":"com.amazonaws.dynamodb.v20120810#ResourceNotFoundException","message":"Requested resource not found"}`,
			wantErr:        true,
			wantErrorType:  "ResourceNotFoundException",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/x-amz-json-1.0")
					w.WriteHeader(tc.responseStatus)
					_, _ = w.Write([]byte(tc.responseBody))
				},
			))
			defer srv.Close()

			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			svc := dynamodb.New(dynamodb.Options{
				Region:             "us-west-2",
				BaseEndpoint:       &srv.URL,
				AuthSchemeResolver: &dynamoDBAuthResolver{},
				AuthSchemes: []smithyhttp.AuthScheme{
					smithyhttp.NewAnonymousScheme(),
				},
				Retryer: aws.NopRetryer{},
			})

			_, err := svc.GetItem(t.Context(), &dynamodb.GetItemInput{
				TableName: aws.String("table1"),
				Key: map[string]dtypes.AttributeValue{
					"id": &dtypes.AttributeValueMemberS{Value: "test"},
				},
			}, func(options *dynamodb.Options) {
				otelaws.AppendMiddlewares(&options.APIOptions, otelaws.WithMeterProvider(mp))
			})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(t.Context(), &rm))
			require.Len(t, rm.ScopeMetrics, 1)

			sm := rm.ScopeMetrics[0]
			assert.Equal(t, instrumentation.Scope{
				Name:    otelaws.ScopeName,
				Version: otelaws.Version,
			}, sm.Scope)
			require.Len(t, sm.Metrics, 1)

			attrs := []attribute.KeyValue{
				attribute.String("db.system.name", "aws.dynamodb"),
				attribute.String("db.operation.name", "GetItem"),
			}
			if tc.wantErrorType != "" {
				attrs = append(attrs, attribute.String("error.type", tc.wantErrorType))
			}

			want := metricdata.Metrics{
				Name:        "db.client.operation.duration",
				Description: "Duration of database client operations.",
				Unit:        "s",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{Attributes: attribute.NewSet(attrs...), Count: 1},
					},
				},
			}
			metricdatatest.AssertEqual(t, want, sm.Metrics[0],
				metricdatatest.IgnoreTimestamp(),
				metricdatatest.IgnoreValue(),
				metricdatatest.IgnoreExemplars(),
			)
		})
	}
}

func TestNonDynamoDBNoClientOperationDuration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<ChangeResourceRecordSetsResponse><ChangeInfo><Id>mockID</Id></ChangeInfo></ChangeResourceRecordSetsResponse>`))
		},
	))
	defer srv.Close()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	svc := route53.New(route53.Options{
		Region:             "us-east-1",
		BaseEndpoint:       &srv.URL,
		AuthSchemeResolver: &route53AuthResolver{},
		AuthSchemes: []smithyhttp.AuthScheme{
			smithyhttp.NewAnonymousScheme(),
		},
		Retryer: aws.NopRetryer{},
	})

	_, err := svc.ChangeResourceRecordSets(t.Context(), &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{},
		},
		HostedZoneId: aws.String("zone"),
	}, func(options *route53.Options) {
		otelaws.AppendMiddlewares(&options.APIOptions, otelaws.WithMeterProvider(mp))
	})
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))
	assert.Empty(t, rm.ScopeMetrics)
}
