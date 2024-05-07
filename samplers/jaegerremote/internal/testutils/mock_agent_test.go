// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jaeger_api_v2 "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/proto-gen/jaeger-idl/proto/api_v2"
	"go.opentelemetry.io/contrib/samplers/jaegerremote/internal/utils"
)

func TestMockAgentSamplingManager(t *testing.T) {
	mockAgent, err := StartMockAgent()
	require.NoError(t, err)
	defer mockAgent.Close()

	err = utils.GetJSON("http://"+mockAgent.SamplingServerAddr()+"/", nil)
	require.Error(t, err, "no 'service' parameter")
	err = utils.GetJSON("http://"+mockAgent.SamplingServerAddr()+"/?service=a&service=b", nil)
	require.Error(t, err, "Too many 'service' parameters")

	var resp jaeger_api_v2.SamplingStrategyResponse
	err = utils.GetJSON("http://"+mockAgent.SamplingServerAddr()+"/?service=something", &resp)
	require.NoError(t, err)
	assert.Equal(t, jaeger_api_v2.SamplingStrategyType_PROBABILISTIC, resp.StrategyType)

	mockAgent.AddSamplingStrategy("service123", &jaeger_api_v2.SamplingStrategyResponse{
		StrategyType: jaeger_api_v2.SamplingStrategyType_RATE_LIMITING,
		RateLimitingSampling: &jaeger_api_v2.RateLimitingSamplingStrategy{
			MaxTracesPerSecond: 123,
		},
	})
	err = utils.GetJSON("http://"+mockAgent.SamplingServerAddr()+"/?service=service123", &resp)
	require.NoError(t, err)
	assert.Equal(t, jaeger_api_v2.SamplingStrategyType_RATE_LIMITING, resp.StrategyType)
	require.NotNil(t, resp.RateLimitingSampling)
	assert.EqualValues(t, 123, resp.RateLimitingSampling.MaxTracesPerSecond)
}
