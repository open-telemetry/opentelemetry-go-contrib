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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/trace"
)

func TestShouldSample(t *testing.T) {
	clock := mockClock{
		nowTime: 1500000000,
	}

	fs := NewFallbackSampler()
	fs.clock = &clock

	sd := fs.ShouldSample(trace.SamplingParameters{})

	assert.Equal(t, trace.RecordAndSample, sd.Decision)
}

func TestBorrowOnePerSecond(t *testing.T) {
	clock := mockClock{
		nowTime: 1500000000,
	}

	fs := NewFallbackSampler()
	fs.clock = &clock

	borrowed := fs.borrow(clock.nowTime)

	// assert that borrowing one per second
	assert.True(t, borrowed)

	borrowed = fs.borrow(clock.nowTime)

	// assert that borrowing again is false during that second
	assert.False(t, borrowed)
}
