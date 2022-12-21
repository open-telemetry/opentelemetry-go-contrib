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

package xray

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	xraypropagator "go.opentelemetry.io/contrib/propagators/aws/xray"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func Test_ShouldSample(t *testing.T) {
	parentCtx := trace.ContextWithSpanContext(
		context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceState: trace.TraceState{},
		}),
	)

	generator := xraypropagator.NewIDGenerator()

	tests := []struct {
		name     string
		fraction float64
	}{
		{
			name:     "should always sample",
			fraction: 1,
		},
		{
			name:     "should nerver sample",
			fraction: 0,
		},
		{
			name:     "should sample 50%",
			fraction: 0.5,
		},
		{
			name:     "should sample 10%",
			fraction: 0.1,
		},
		{
			name:     "should sample 1%",
			fraction: 0.01,
		},
	}

	totalIterations := 100000
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalSampled := 0
			s := NewTraceIDRatioBased(tt.fraction)
			for i := 0; i < totalIterations; i++ {
				traceID, _ := generator.NewIDs(context.Background())
				r := s.ShouldSample(
					sdktrace.SamplingParameters{
						ParentContext: parentCtx,
						TraceID:       traceID,
						Name:          "test",
						Kind:          trace.SpanKindServer,
					})
				if r.Decision == sdktrace.RecordAndSample {
					totalSampled++
				}
			}

			tolerance := 0.1
			expected := tt.fraction * float64(totalIterations)
			require.InDelta(t, expected, totalSampled, expected*tolerance)
		})
	}
}
