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

package otelsql

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
)

func TestRecordSpanError(t *testing.T) {
	testCases := []struct {
		name          string
		opts          SpanOptions
		err           error
		expectedError bool
	}{
		{
			name:          "no error",
			err:           nil,
			expectedError: false,
		},
		{
			name:          "normal error",
			err:           errors.New("error"),
			expectedError: true,
		},
		{
			name:          "normal error with DisableErrSkip",
			err:           errors.New("error"),
			opts:          SpanOptions{DisableErrSkip: true},
			expectedError: true,
		},
		{
			name:          "ErrSkip error",
			err:           driver.ErrSkip,
			expectedError: true,
		},
		{
			name:          "ErrSkip error with DisableErrSkip",
			err:           driver.ErrSkip,
			opts:          SpanOptions{DisableErrSkip: true},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var span oteltest.Span
			recordSpanError(&span, tc.opts, tc.err)

			if tc.expectedError {
				assert.Equal(t, codes.Error, span.StatusCode())
			} else {
				assert.Equal(t, codes.Unset, span.StatusCode())
			}
		})
	}
}

func newTracerProvider() (*oteltest.StandardSpanRecorder, *oteltest.TracerProvider) {
	var sr oteltest.StandardSpanRecorder
	provider := oteltest.NewTracerProvider(
		oteltest.WithSpanRecorder(&sr),
	)
	return &sr, provider
}

func createDummySpan(ctx context.Context, tracer trace.Tracer) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "dummy")
	defer span.End()
	return ctx, span
}

func newMockConfig(tracer trace.Tracer) config {
	return config{
		Tracer:            tracer,
		Attributes:        []label.KeyValue{defaultLabel},
		SpanNameFormatter: &defaultSpanNameFormatter{},
	}
}

func attributesListToMap(labels []label.KeyValue) map[label.Key]label.Value {
	attributes := make(map[label.Key]label.Value)

	for _, v := range labels {
		attributes[v.Key] = v.Value
	}
	return attributes
}
