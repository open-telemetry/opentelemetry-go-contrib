// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
)

func FuzzParseAndUpdate(f *testing.F) {
	seeds := []string{
		`{"strategyType":0,"probabilisticSampling":{"samplingRate":0.5}}`,
		`{"strategyType":"PROBABILISTIC","probabilisticSampling":{"samplingRate":0.25}}`,
		`{"strategyType":1,"rateLimitingSampling":{"maxTracesPerSecond":10}}`,
		`{"strategyType":2,"operationSampling":{"defaultSamplingProbability":0.5,"defaultLowerBoundTracesPerSecond":1.0,"perOperationStrategies":[{"operation":"GET /users","probabilisticSampling":{"samplingRate":0.8}}]}}`,
		`{"operationSampling":{"defaultSamplingProbability":0.5,"perOperationStrategies":[{"operation":"GET /health"},null]}}`,
		`{"operationSampling":{}}`,
		`{}`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(_ *testing.T, data []byte) {
		parser := &samplingStrategyParserImpl{}
		strategy, err := parser.Parse(data)
		if err != nil {
			return
		}

		updaters := []samplerUpdater{
			&probabilisticSamplerUpdater{},
			&rateLimitingSamplerUpdater{},
			&perOperationSamplerUpdater{
				MaxOperations:            defaultSamplingMaxOperations,
				OperationNameLateBinding: defaultSamplingOperationNameLateBinding,
			},
		}

		var current trace.Sampler = newProbabilisticSampler(0.5, false, false)
		for range 2 {
			for _, updater := range updaters {
				modified, updateErr := updater.Update(current, strategy)
				if updateErr != nil {
					continue
				}
				if modified != nil {
					current = modified
					break
				}
			}
		}
	})
}
