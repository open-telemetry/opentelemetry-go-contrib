// Copyright The OpenTelemetry Authors
// Copyright (c) 2021 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
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

package jaegerremote

import (
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/contrib/samplers/jaegerremote/internal/testutils"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestRemotelyControlledSampler_updateRace(t *testing.T) {
	initSampler := newProbabilisticSampler(0.123)
	fetcher := &testSamplingStrategyFetcher{response: []byte("probabilistic")}
	parser := new(testSamplingStrategyParser)
	updaters := []samplerUpdater{new(probabilisticSamplerUpdater)}
	sampler := New(
		"test",
		WithMaxOperations(42),
		WithOperationNameLateBinding(true),
		WithInitialSampler(initSampler),
		WithSamplingServerURL("my url"),
		WithSamplingRefreshInterval(time.Millisecond),
		withSamplingStrategyFetcher(fetcher),
		withSamplingStrategyParser(parser),
		withUpdaters(updaters...),
	)

	s := makeSamplingParameters(1, "test")
	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	go accessor(func() {
		sampler.UpdateSampler()
	})

	go accessor(func() {
		sampler.ShouldSample(s)
	})

	time.Sleep(100 * time.Millisecond)
	close(end)
	sampler.Close()
}

type testSamplingStrategyFetcher struct {
	response []byte
}

func (c *testSamplingStrategyFetcher) Fetch(serviceName string) ([]byte, error) {
	return c.response, nil
}

type testSamplingStrategyParser struct {
}

func (p *testSamplingStrategyParser) Parse(response []byte) (interface{}, error) {
	strategy := new(jaeger_api_v2.SamplingStrategyResponse)

	switch string(response) {
	case "probabilistic":
		strategy.StrategyType = jaeger_api_v2.SamplingStrategyType_PROBABILISTIC
		strategy.ProbabilisticSampling = &jaeger_api_v2.ProbabilisticSamplingStrategy{
			SamplingRate: 0.85,
		}
		return strategy, nil
	case "rateLimiting":
		strategy.StrategyType = jaeger_api_v2.SamplingStrategyType_RATE_LIMITING
		strategy.RateLimitingSampling = &jaeger_api_v2.RateLimitingSamplingStrategy{
			MaxTracesPerSecond: 100,
		}
		return strategy, nil
	}

	return nil, errors.New("unknown strategy test request")
}

func TestRemoteSamplerOptions(t *testing.T) {
	initSampler := newProbabilisticSampler(0.123)
	fetcher := new(fakeSamplingFetcher)
	parser := new(samplingStrategyParserImpl)
	updaters := []samplerUpdater{new(probabilisticSamplerUpdater)}
	sampler := New(
		"test",
		WithMaxOperations(42),
		WithOperationNameLateBinding(true),
		WithInitialSampler(initSampler),
		WithSamplingServerURL("my url"),
		WithSamplingRefreshInterval(42*time.Second),
		withSamplingStrategyFetcher(fetcher),
		withSamplingStrategyParser(parser),
		withUpdaters(updaters...),
	)
	assert.Equal(t, 42, sampler.posParams.MaxOperations)
	assert.True(t, sampler.posParams.OperationNameLateBinding)
	assert.Same(t, initSampler, sampler.sampler)
	assert.Equal(t, "my url", sampler.samplingServerURL)
	assert.Equal(t, 42*time.Second, sampler.samplingRefreshInterval)
	assert.Same(t, fetcher, sampler.samplingFetcher)
	assert.Same(t, parser, sampler.samplingParser)
	assert.EqualValues(t, sampler.updaters[0], &perOperationSamplerUpdater{MaxOperations: 42, OperationNameLateBinding: true})
}

func TestRemoteSamplerOptionsDefaults(t *testing.T) {
	options := newConfig()
	sampler, ok := options.sampler.(*probabilisticSampler)
	assert.True(t, ok)
	assert.Equal(t, 0.001, sampler.samplingRate)

	assert.NotEmpty(t, options.samplingServerURL)
	assert.NotZero(t, options.samplingRefreshInterval)
}

func initAgent(t *testing.T) (*testutils.MockAgent, *Sampler) {
	agent, err := testutils.StartMockAgent()
	require.NoError(t, err)

	initialSampler := newProbabilisticSampler(0.001)
	sampler := New(
		"client app",
		WithSamplingServerURL("http://"+agent.SamplingServerAddr()),
		WithMaxOperations(testDefaultMaxOperations),
		WithInitialSampler(initialSampler),
		WithSamplingRefreshInterval(time.Minute),
	)
	sampler.Close() // stop timer-based updates, we want to call them manually

	return agent, sampler
}

func makeSamplingParameters(id uint64, operationName string) trace.SamplingParameters {
	var traceID oteltrace.TraceID
	binary.BigEndian.PutUint64(traceID[:], id)

	return trace.SamplingParameters{
		TraceID: traceID,
		Name:    operationName,
	}
}

func TestRemotelyControlledSampler(t *testing.T) {
	agent, remoteSampler := initAgent(t)
	defer agent.Close()

	defaultSampler := newProbabilisticSampler(0.001)
	remoteSampler.setSampler(defaultSampler)

	agent.AddSamplingStrategy("client app",
		getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, testDefaultSamplingProbability))
	remoteSampler.UpdateSampler()
	s1, ok := remoteSampler.sampler.(*probabilisticSampler)
	assert.True(t, ok)
	assert.EqualValues(t, testDefaultSamplingProbability, s1.samplingRate, "Sampler should have been updated")

	result := remoteSampler.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
	assert.Equal(t, trace.Drop, result.Decision)
	result = remoteSampler.ShouldSample(makeSamplingParameters(testMaxID-10, testOperationName))
	assert.Equal(t, trace.RecordAndSample, result.Decision)

	remoteSampler.setSampler(defaultSampler)

	c := make(chan time.Time)
	ticker := &time.Ticker{C: c}
	// reset closed so the next call to Close() correctly stops the polling goroutine
	remoteSampler.closed = 0
	go remoteSampler.pollControllerWithTicker(ticker)

	c <- time.Now() // force update based on timer
	time.Sleep(10 * time.Millisecond)
	remoteSampler.Close()

	s2, ok := remoteSampler.sampler.(*probabilisticSampler)
	assert.True(t, ok)
	assert.EqualValues(t, testDefaultSamplingProbability, s2.samplingRate, "Sampler should have been updated from timer")
}

func TestRemotelyControlledSampler_updateSampler(t *testing.T) {
	tests := []struct {
		probabilities              map[string]float64
		defaultProbability         float64
		expectedDefaultProbability float64
	}{
		{
			probabilities:              map[string]float64{testOperationName: 1.1},
			defaultProbability:         testDefaultSamplingProbability,
			expectedDefaultProbability: testDefaultSamplingProbability,
		},
		{
			probabilities:              map[string]float64{testOperationName: testDefaultSamplingProbability},
			defaultProbability:         testDefaultSamplingProbability,
			expectedDefaultProbability: testDefaultSamplingProbability,
		},
		{
			probabilities: map[string]float64{
				testOperationName:          testDefaultSamplingProbability,
				testFirstTimeOperationName: testDefaultSamplingProbability,
			},
			defaultProbability:         testDefaultSamplingProbability,
			expectedDefaultProbability: testDefaultSamplingProbability,
		},
		{
			probabilities:              map[string]float64{"new op": 1.1},
			defaultProbability:         testDefaultSamplingProbability,
			expectedDefaultProbability: testDefaultSamplingProbability,
		},
		{
			probabilities:              map[string]float64{"new op": 1.1},
			defaultProbability:         1.1,
			expectedDefaultProbability: 1.0,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			agent, sampler := initAgent(t)
			defer agent.Close()

			initSampler, ok := sampler.sampler.(*probabilisticSampler)
			assert.True(t, ok)

			res := &jaeger_api_v2.SamplingStrategyResponse{
				StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
				OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
					DefaultSamplingProbability:       test.defaultProbability,
					DefaultLowerBoundTracesPerSecond: 0.001,
				},
			}
			for opName, prob := range test.probabilities {
				res.OperationSampling.PerOperationStrategies = append(res.OperationSampling.PerOperationStrategies,
					&jaeger_api_v2.OperationSamplingStrategy{
						Operation: opName,
						ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
							SamplingRate: prob,
						},
					},
				)
			}

			agent.AddSamplingStrategy("client app", res)
			sampler.UpdateSampler()

			s, ok := sampler.sampler.(*perOperationSampler)
			assert.True(t, ok)
			assert.NotEqual(t, initSampler, sampler.sampler, "Sampler should have been updated")
			assert.Equal(t, test.expectedDefaultProbability, s.defaultSampler.SamplingRate())

			// First call is always sampled
			result := sampler.ShouldSample(makeSamplingParameters(testMaxID+10, testOperationName))
			assert.Equal(t, trace.RecordAndSample, result.Decision)

			result = sampler.ShouldSample(makeSamplingParameters(testMaxID-10, testOperationName))
			assert.Equal(t, trace.RecordAndSample, result.Decision)
		})
	}
}

func TestRemotelyControlledSampler_ImmediatelyUpdateOnStartup(t *testing.T) {
	initSampler := newProbabilisticSampler(0.123)
	fetcher := &testSamplingStrategyFetcher{response: []byte("rateLimiting")}
	parser := new(testSamplingStrategyParser)
	updaters := []samplerUpdater{new(probabilisticSamplerUpdater), new(rateLimitingSamplerUpdater)}
	sampler := New(
		"test",
		WithMaxOperations(42),
		WithOperationNameLateBinding(true),
		WithInitialSampler(initSampler),
		WithSamplingServerURL("my url"),
		WithSamplingRefreshInterval(10*time.Minute),
		withSamplingStrategyFetcher(fetcher),
		withSamplingStrategyParser(parser),
		withUpdaters(updaters...),
	)
	time.Sleep(100 * time.Millisecond) // waiting for s.pollController
	sampler.Close()                    // stop pollController, avoid date race
	s, ok := sampler.sampler.(*rateLimitingSampler)
	assert.True(t, ok)
	assert.Equal(t, float64(100), s.maxTracesPerSecond)
}

func TestRemotelyControlledSampler_multiStrategyResponse(t *testing.T) {
	agent, sampler := initAgent(t)
	defer agent.Close()
	initSampler, ok := sampler.sampler.(*probabilisticSampler)
	assert.True(t, ok)

	defaultSampingRate := 1.0
	testUnusedOpName := "unused_op"
	testUnusedOpSamplingRate := 0.0

	res := &jaeger_api_v2.SamplingStrategyResponse{
		StrategyType:          jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
		ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{SamplingRate: defaultSampingRate},
		OperationSampling: &jaeger_api_v2.PerOperationSamplingStrategies{
			DefaultSamplingProbability:       defaultSampingRate,
			DefaultLowerBoundTracesPerSecond: 0.001,
			PerOperationStrategies: []*jaeger_api_v2.OperationSamplingStrategy{
				{
					Operation: testUnusedOpName,
					ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
						SamplingRate: testUnusedOpSamplingRate,
					}},
			},
		},
	}

	agent.AddSamplingStrategy("client app", res)
	sampler.UpdateSampler()
	s, ok := sampler.sampler.(*perOperationSampler)
	assert.True(t, ok)
	assert.NotEqual(t, initSampler, sampler.sampler, "Sampler should have been updated")
	assert.Equal(t, defaultSampingRate, s.defaultSampler.SamplingRate())

	result := sampler.ShouldSample(makeSamplingParameters(testMaxID-10, testUnusedOpName))
	assert.Equal(t, trace.RecordAndSample, result.Decision) // first call always pass
	result = sampler.ShouldSample(makeSamplingParameters(testMaxID, testUnusedOpName))
	assert.Equal(t, trace.Drop, result.Decision)
}

func TestSamplerQueryError(t *testing.T) {
	agent, sampler := initAgent(t)
	defer agent.Close()

	// override the actual handler
	sampler.samplingFetcher = &fakeSamplingFetcher{}

	initSampler, ok := sampler.sampler.(*probabilisticSampler)
	assert.True(t, ok)

	sampler.Close() // stop timer-based updates, we want to call them manually

	sampler.UpdateSampler()
	assert.Equal(t, initSampler, sampler.sampler, "Sampler should not have been updated due to query error")
}

type fakeSamplingFetcher struct{}

func (c *fakeSamplingFetcher) Fetch(serviceName string) ([]byte, error) {
	return nil, errors.New("query error")
}

func TestRemotelyControlledSampler_updateSamplerFromAdaptiveSampler(t *testing.T) {
	agent, remoteSampler := initAgent(t)
	defer agent.Close()
	remoteSampler.Close() // close the second time (initAgent already called Close)

	strategies := &jaeger_api_v2.PerOperationSamplingStrategies{
		DefaultSamplingProbability:       testDefaultSamplingProbability,
		DefaultLowerBoundTracesPerSecond: 1.0,
	}
	adaptiveSampler := newPerOperationSampler(perOperationSamplerParams{
		MaxOperations: testDefaultMaxOperations,
		Strategies:    strategies,
	})

	// Overwrite the sampler with an adaptive sampler
	remoteSampler.setSampler(adaptiveSampler)

	agent.AddSamplingStrategy("client app",
		getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, 0.5))
	remoteSampler.UpdateSampler()

	// Sampler should have been updated to probabilistic
	_, ok := remoteSampler.sampler.(*probabilisticSampler)
	require.True(t, ok)

	// Overwrite the sampler with an adaptive sampler
	remoteSampler.setSampler(adaptiveSampler)

	agent.AddSamplingStrategy("client app",
		getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_RATE_LIMITING, 1))
	remoteSampler.UpdateSampler()

	// Sampler should have been updated to ratelimiting
	_, ok = remoteSampler.sampler.(*rateLimitingSampler)
	require.True(t, ok)

	// Overwrite the sampler with an adaptive sampler
	remoteSampler.setSampler(adaptiveSampler)

	// Update existing adaptive sampler
	agent.AddSamplingStrategy("client app", &jaeger_api_v2.SamplingStrategyResponse{OperationSampling: strategies})
	remoteSampler.UpdateSampler()
}

func TestRemotelyControlledSampler_updateRateLimitingOrProbabilisticSampler(t *testing.T) {
	probabilisticSampler := newProbabilisticSampler(0.002)
	otherProbabilisticSampler := newProbabilisticSampler(0.003)
	maxProbabilisticSampler := newProbabilisticSampler(1.0)

	rateLimitingSampler := newRateLimitingSampler(2)
	otherRateLimitingSampler := newRateLimitingSampler(3)

	testCases := []struct {
		res                  *jaeger_api_v2.SamplingStrategyResponse
		initSampler          trace.Sampler
		expectedSampler      trace.Sampler
		shouldErr            bool
		referenceEquivalence bool
		caption              string
	}{
		{
			res:                  getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, 1.5),
			initSampler:          probabilisticSampler,
			expectedSampler:      maxProbabilisticSampler,
			shouldErr:            true,
			referenceEquivalence: false,
			caption:              "invalid probabilistic strategy",
		},
		{
			res:                  getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, 0.002),
			initSampler:          probabilisticSampler,
			expectedSampler:      probabilisticSampler,
			shouldErr:            false,
			referenceEquivalence: true,
			caption:              "unchanged probabilistic strategy",
		},
		{
			res:                  getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, 0.003),
			initSampler:          probabilisticSampler,
			expectedSampler:      otherProbabilisticSampler,
			shouldErr:            false,
			referenceEquivalence: false,
			caption:              "valid probabilistic strategy",
		},
		{
			res:                  getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_RATE_LIMITING, 2),
			initSampler:          rateLimitingSampler,
			expectedSampler:      rateLimitingSampler,
			shouldErr:            false,
			referenceEquivalence: true,
			caption:              "unchanged rate limiting strategy",
		},
		{
			res:                  getSamplingStrategyResponse(jaeger_api_v2.SamplingStrategyType_RATE_LIMITING, 3),
			initSampler:          rateLimitingSampler,
			expectedSampler:      otherRateLimitingSampler,
			shouldErr:            false,
			referenceEquivalence: false,
			caption:              "valid rate limiting strategy",
		},
		{
			res:                  &jaeger_api_v2.SamplingStrategyResponse{},
			initSampler:          rateLimitingSampler,
			expectedSampler:      rateLimitingSampler,
			shouldErr:            true,
			referenceEquivalence: true,
			caption:              "invalid strategy",
		},
	}

	for _, tc := range testCases {
		testCase := tc // capture loop var
		t.Run(testCase.caption, func(t *testing.T) {
			remoteSampler := New(
				"test",
				WithInitialSampler(testCase.initSampler),
				withUpdaters(
					new(probabilisticSamplerUpdater),
					new(rateLimitingSamplerUpdater),
				),
			)
			err := remoteSampler.updateSamplerViaUpdaters(testCase.res)
			if testCase.shouldErr {
				require.Error(t, err)
				return
			}
			if testCase.referenceEquivalence {
				assert.Equal(t, testCase.expectedSampler, remoteSampler.sampler)
			} else {
				type comparable interface {
					Equal(other trace.Sampler) bool
				}
				es, esOk := testCase.expectedSampler.(comparable)
				require.True(t, esOk, "expected sampler %+v must implement Equal()", testCase.expectedSampler)
				assert.True(t, es.Equal(remoteSampler.sampler),
					"sampler.Equal: want=%+v, have=%+v", testCase.expectedSampler, remoteSampler.sampler)
			}
		})
	}
}

func getSamplingStrategyResponse(strategyType jaeger_api_v2.SamplingStrategyType, value float64) *jaeger_api_v2.SamplingStrategyResponse {
	if strategyType == jaeger_api_v2.SamplingStrategyType_PROBABILISTIC {
		return &jaeger_api_v2.SamplingStrategyResponse{
			StrategyType: jaeger_api_v2.SamplingStrategyType_PROBABILISTIC,
			ProbabilisticSampling: &jaeger_api_v2.ProbabilisticSamplingStrategy{
				SamplingRate: value,
			},
		}
	}
	if strategyType == jaeger_api_v2.SamplingStrategyType_RATE_LIMITING {
		return &jaeger_api_v2.SamplingStrategyResponse{
			StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
			RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
				MaxTracesPerSecond: int32(value),
			},
		}
	}
	return nil
}
