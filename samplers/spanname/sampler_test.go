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

package spanname

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestIgnoreSpansWithNameSampler(t *testing.T) {
	testCases := []struct {
		name     string
		filter   string
		decision sdktrace.SamplingDecision
		tsStr    string
	}{
		{
			"grpc.health.v1.Health/Check",
			"grpc.health.v1.Health",
			sdktrace.Drop,
			"k=v",
		},
		{
			"grpc.health.v1.Health/Check",
			"example.span",
			sdktrace.RecordAndSample,
			"x=y",
		},
	}

	for _, tc := range testCases {
		traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
		ts, err := trace.ParseTraceState(tc.tsStr)
		if err != nil {
			t.Errorf("failed to parse test case TraceState: %v", err)
		}
		pc := trace.ContextWithSpanContext(
			context.Background(),
			trace.NewSpanContext(trace.SpanContextConfig{
				TraceState: ts,
			}),
		)
		sp := sdktrace.SamplingParameters{
			TraceID:       traceID,
			Name:          tc.name,
			ParentContext: pc,
		}

		r := IgnoreSpansWithNameSampler(tc.filter).ShouldSample(sp)
		require.Equal(t, tc.decision, r.Decision,
			"wanted %v but %v obtained", tc.decision, r.Decision)

		require.Equal(t, ts, r.Tracestate,
			"wanted %v but %v obtained", ts, r.Tracestate)
	}
}
