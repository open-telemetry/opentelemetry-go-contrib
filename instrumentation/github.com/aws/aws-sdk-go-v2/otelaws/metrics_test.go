// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"errors"
	"fmt"
	"testing"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewDefaultMetricRecorder(t *testing.T) {
	recorder := NewDefaultMetricRecorder(noop.NewMeterProvider().Meter(""))
	require.NotNil(t, recorder)

	ctx := v2Middleware.SetServiceID(t.Context(), dynamodb.ServiceID)
	assert.NotPanics(t, func() {
		recorder(ctx, middleware.InitializeInput{}, middleware.InitializeOutput{}, nil)
	})
}

func TestErrorTypeAttr(t *testing.T) {
	apiErr := &smithy.GenericAPIError{Code: "ResourceNotFoundException"}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "API error", err: apiErr, want: "ResourceNotFoundException"},
		{
			name: "wrapped API error",
			err:  &smithy.OperationError{ServiceID: "DynamoDB", OperationName: "GetItem", Err: fmt.Errorf("wrapped: %w", apiErr)},
			want: "ResourceNotFoundException",
		},
		{name: "API error without code", err: &smithy.GenericAPIError{}, want: "*smithy.GenericAPIError"},
		{name: "non-API error", err: errors.New("boom"), want: "*errors.errorString"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorTypeAttr(tt.err)
			assert.Equal(t, "error.type", string(got.Key))
			assert.Equal(t, tt.want, got.Value.AsString())
		})
	}
}
