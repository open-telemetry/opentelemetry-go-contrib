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

package xrayconfig

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel/trace"
)

func TestEventToCarrier(t *testing.T) {
	os.Clearenv()

	_ = os.Setenv("_X_AMZN_TRACE_ID", "traceID")
	carrier := xrayEventToCarrier([]byte{})

	assert.Equal(t, "traceID", carrier.Get("X-Amzn-Trace-Id"))
}

func TestEventToCarrierWithPropagator(t *testing.T) {
	os.Clearenv()

	_ = os.Setenv("_X_AMZN_TRACE_ID", "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
	carrier := xrayEventToCarrier([]byte{})
	ctx := xray.Propagator{}.Extract(context.Background(), carrier)

	expectedTraceID, _ := trace.TraceIDFromHex("5759e988bd862e3fe1be46a994272793")
	expectedSpanID, _ := trace.SpanIDFromHex("53995c3f42cd8ad8")
	expectedCtx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    expectedTraceID,
		SpanID:     expectedSpanID,
		TraceFlags: trace.FlagsSampled,
		TraceState: trace.TraceState{},
		Remote:     true,
	}))

	assert.Equal(t, expectedCtx, ctx)
}
