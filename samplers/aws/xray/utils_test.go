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
	"time"

	"github.com/stretchr/testify/assert"
)

// MockClock is a struct to record current time.
type mockClock struct {
	nowTime  int64
	nowNanos int64
}

// Now function returns NowTime value.
func (c *mockClock) now() time.Time {
	return time.Unix(c.nowTime, c.nowNanos)
}

func TestNewTicker(t *testing.T) {
	ticker := newTicker(300*time.Second, 5*time.Second)

	assert.Equal(t, ticker.d, 5*time.Minute)
	assert.NotEmpty(t, ticker.t)
	assert.NotEmpty(t, ticker.jitter)
}
