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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockClock is a struct to record current time.
type MockClock struct {
	NowTime  int64
	NowNanos int64
}

// Now function returns NowTime value.
func (c *MockClock) Now() time.Time {
	return time.Unix(c.NowTime, c.NowNanos)
}

// Increment is a method to increase current time.
func (c *MockClock) Increment(s int64, ns int64) time.Time {
	sec := atomic.AddInt64(&c.NowTime, s)
	nSec := atomic.AddInt64(&c.NowNanos, ns)

	return time.Unix(sec, nSec)
}

func TestNewTicker(t *testing.T) {
	ticker := newTicker(300*time.Second, 5*time.Second)

	assert.Equal(t, ticker.d, 5*time.Minute)
	assert.NotEmpty(t, ticker.t)
	assert.NotEmpty(t, ticker.jitter)
}
