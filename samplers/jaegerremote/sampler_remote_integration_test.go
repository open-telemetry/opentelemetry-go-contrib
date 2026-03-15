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

func TestRemotelyControlledSampler_Attributes(t *testing.T) {
	agent, err := testutils.StartMockAgent()
	require.NoError(t, err)

	remoteSampler := jaegerremote.New(
		"client app",
		jaegerremote.WithSamplingServerURL("http://"+agent.SamplingServerAddr()),
		jaegerremote.WithSamplingRefreshInterval(time.Minute),
	)
	remoteSampler.Close() // stop timer-based updates, we want to call them manually
	defer agent.Close()

	var traceID oteltrace.TraceID
	binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)

	t.Run("probabilistic", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: testDefaultSamplingProbability,
				},
			})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String("jaeger.sampler.type", "probabilistic"), attribute.Float64("jaeger.sampler.param", 0.5)}, result.Attributes)
	})

	t.Run("ratelimitng", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
				RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
					MaxTracesPerSecond: 1,
				},
			})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String("jaeger.sampler.type", "ratelimiting"), attribute.Float64("jaeger.sampler.param", 1)}, result.Attributes)
	})

	t.Run("per operation", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
				DefaultSamplingProbability:       testDefaultSamplingProbability,
				DefaultLowerBoundTracesPerSecond: 1.0,
			}})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Equal(t, []attribute.KeyValue{attribute.String("jaeger.sampler.type", "probabilistic"), attribute.Float64("jaeger.sampler.param", 0.5)}, result.Attributes)
	})
}

func TestRemotelyControlledSampler_AttributesDisabled(t *testing.T) {
	agent, err := testutils.StartMockAgent()
	require.NoError(t, err)

	remoteSampler := jaegerremote.New(
		"client app",
		jaegerremote.WithSamplingServerURL("http://"+agent.SamplingServerAddr()),
		jaegerremote.WithSamplingRefreshInterval(time.Minute),
		jaegerremote.WithAttributesDisabled(),
	)
	remoteSampler.Close() // stop timer-based updates, we want to call them manually
	defer agent.Close()

	var traceID oteltrace.TraceID
	binary.BigEndian.PutUint64(traceID[8:], testMaxID-20)

	t.Run("probabilistic", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
					SamplingRate: testDefaultSamplingProbability,
				},
			})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
	})

	t.Run("ratelimitng", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
				RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
					MaxTracesPerSecond: 1,
				},
			})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
	})

	t.Run("per operation", func(t *testing.T) {
		agent.AddSamplingStrategy("client app",
			&jaeger_api_v2.SamplingStrategyResponse{OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
				DefaultSamplingProbability:       testDefaultSamplingProbability,
				DefaultLowerBoundTracesPerSecond: 1.0,
			}})
		remoteSampler.UpdateSampler()

		result := remoteSampler.ShouldSample(trace.SamplingParameters{TraceID: traceID})
		assert.Equal(t, trace.RecordAndSample, result.Decision)
		assert.Nil(t, result.Attributes)
	})
}
