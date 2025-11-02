// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote_test

import (
	"encoding/binary"
	"testing"
	"time"

	jaeger_api_v2 "github.com/jaegertracing/jaeger-idl/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/contrib/samplers/jaegerremote/internal/testutils"
)

const (
	testDefaultSamplingProbability = 0.5
	testMaxID                      = uint64(1) << 63
)

func TestRemotelyControlledSampler_WithAttributesOn(t *testing.T) {
	agent, err := testutils.StartMockAgent()
	require.NoError(t, err)

	remoteSampler := jaegerremote.New(
		"client app",
		jaegerremote.WithSamplingServerURL("http://"+agent.SamplingServerAddr()),
		jaegerremote.WithSamplingRefreshInterval(time.Minute),
		jaegerremote.WithAttributesOn(),
	)
	remoteSampler.Close() // stop timer-based updates, we want to call them manually
	defer agent.Close()

	var traceID oteltrace.TraceID
	binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)

	// Probabilistic
	agent.AddSamplingStrategy("client app",
		&jaeger_api_v2.SamplingStrategyResponse{
			StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
				SamplingRate: testDefaultSamplingProbability,
			},
		},
	)
	remoteSampler.UpdateSampler()

	result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
	assert.Equal(t, trace.RecordAndSample, result.Decision)
	assert.Equal(t, []attribute.KeyValue{attribute.String("sampler.type", "probabilistic"), attribute.Float64("sampler.param", 0.5)}, result.Attributes)
}
