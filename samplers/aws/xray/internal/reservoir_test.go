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

package internal

import (
	"testing"

	"go.opentelemetry.io/contrib/samplers/aws/xray/internal/util"

	"github.com/stretchr/testify/assert"
)

// assert that reservoir quota is expired.
func TestExpiredReservoir(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000001,
	}

	r := &reservoir{
		expiresAt: 1500000000,
	}

	expired := r.expired(clock.Now().Unix())

	assert.True(t, expired)
}

// assert that reservoir quota is still expired since now time is equal to expiresAt time.
func TestExpiredReservoirSameAsClockTime(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		expiresAt: 1500000000,
	}

	expired := r.expired(clock.Now().Unix())

	assert.True(t, expired)
}

// Assert that borrow only 1 req/sec
func TestBorrowEverySecond(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		capacity: 10,
	}

	s := r.borrow(clock.Now().Unix())
	assert.True(t, s)

	s = r.borrow(clock.Now().Unix())
	assert.False(t, s)

	// Increment clock by 1
	clock = &util.MockClock{
		NowTime: 1500000001,
	}

	s = r.borrow(clock.Now().Unix())
	assert.True(t, s)
}

// assert that we can still borrowing from reservoir is possible since assigned quota is available to consume
// and it will increase used count.
func TestConsumeAvailableQuota(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		quota:        int64(9),
		capacity:     int64(100),
		used:         int64(0),
		currentEpoch: clock.Now().Unix(),
	}

	s := r.take(clock.Now().Unix())
	assert.True(t, s)
	assert.Equal(t, int64(1), r.used)
}

// assert that we can not borrow from reservoir since assigned quota is not available to consume
// and it will not increase used count.
func TestConsumeUnavailableQuota(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		quota:        int64(9),
		capacity:     int64(100),
		used:         int64(9),
		currentEpoch: clock.Now().Unix(),
	}

	s := r.take(clock.Now().Unix())
	assert.False(t, s)
	assert.Equal(t, int64(9), r.used)
}

func TestResetQuotaUsageRotation(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		quota:        int64(5),
		capacity:     int64(100),
		used:         int64(0),
		currentEpoch: clock.Now().Unix(),
	}

	// consume quota for second
	for i := 0; i < 5; i++ {
		taken := r.take(clock.Now().Unix())
		assert.Equal(t, true, taken)
		assert.Equal(t, int64(i+1), r.used)
	}

	// take() should be false since no unused quota left
	taken := r.take(clock.Now().Unix())
	assert.Equal(t, false, taken)
	assert.Equal(t, int64(5), r.used)

	// increment epoch to reset unused quota
	clock = &util.MockClock{
		NowTime: 1500000001,
	}

	// take() should be true since ununsed quota is available
	taken = r.take(clock.Now().Unix())
	assert.Equal(t, int64(1500000001), r.currentEpoch)
	assert.Equal(t, true, taken)
	assert.Equal(t, int64(1), r.used)
}
