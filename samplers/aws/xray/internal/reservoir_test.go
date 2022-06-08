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
	"time"

	"github.com/stretchr/testify/assert"
)

// assert that reservoir quota is expired.
func TestExpiredReservoir(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000001,
	}

	expiresAt := time.Unix(1500000000, 0)
	r := &reservoir{
		expiresAt: expiresAt,
	}

	expired := r.expired(clock.now())

	assert.True(t, expired)
}

// assert that reservoir quota is still expired since now time is equal to expiresAt time.
func TestExpiredReservoirSameAsClockTime(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	expiresAt := time.Unix(1500000000, 0)

	r := &reservoir{
		expiresAt: expiresAt,
	}

	assert.False(t, r.expired(clock.now()))
}

// assert that borrow only 1 req/sec.
func TestBorrowEverySecond(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	r := &reservoir{
		capacity: 10,
	}

	s := r.take(clock.now(), true, 1.0)
	assert.True(t, s)

	s = r.take(clock.now(), true, 1.0)
	assert.False(t, s)

	// Increment clock by 1
	clock = &mockClock{
		nowTime: 1500000001,
	}

	s = r.take(clock.now(), true, 1.0)
	assert.True(t, s)
}

// assert that when reservoir is expired we consume from quota is 1 and then
// when reservoir is not expired consume from assigned quota by X-Ray service.
func TestConsumeFromBorrowConsumeFromQuota(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	r := &reservoir{
		quota:    2,
		capacity: 10,
	}

	s := r.take(clock.now(), true, 1.0)
	assert.True(t, s)

	s = r.take(clock.now(), true, 1.0)
	assert.False(t, s)

	// Increment clock by 1
	clock = &mockClock{
		nowTime: 1500000001,
	}

	s = r.take(clock.now(), true, 1.0)
	assert.True(t, s)

	// Increment clock by 1
	clock = &mockClock{
		nowTime: 1500000002,
	}

	s = r.take(clock.now(), false, 1.0)
	assert.True(t, s)

	s = r.take(clock.now(), false, 1.0)
	assert.True(t, s)

	s = r.take(clock.now(), false, 1.0)
	assert.False(t, s)
}

// assert that we can still borrowing from reservoir is possible since assigned quota is available to consume
// and it will increase used count.
func TestConsumeFromReservoir(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	r := &reservoir{
		quota:    2,
		capacity: 100,
	}

	// reservoir updates the quotaBalance for new second and allows to consume
	// quota balance is 0 because we are consuming from reservoir for the first time
	assert.Equal(t, r.quotaBalance, 0.0)
	assert.True(t, r.take(clock.now(), false, 1.0))
	assert.Equal(t, r.quotaBalance, 1.0)
	assert.True(t, r.take(clock.now(), false, 1.0))
	assert.Equal(t, r.quotaBalance, 0.0)
	// once assigned quota is consumed reservoir does not allow to consume in that second
	assert.False(t, r.take(clock.now(), false, 1.0))

	// increase the clock by 1
	clock.nowTime = 1500000001

	// reservoir updates the quotaBalance for new second and allows to consume
	assert.Equal(t, r.quotaBalance, 0.0)
	assert.True(t, r.take(clock.now(), false, 1.0))
	assert.Equal(t, r.quotaBalance, 1.0)
	assert.True(t, r.take(clock.now(), false, 1.0))
	assert.Equal(t, r.quotaBalance, 0.0)
	// once assigned quota is consumed reservoir does not allow to consume in that second
	assert.False(t, r.take(clock.now(), false, 1.0))

	// increase the clock by 5
	clock.nowTime = 1500000005

	// elapsedTime is 4 seconds so quota balance should be elapsedTime * quota = 8 and below take would consume 1 so
	// ultimately 7
	assert.True(t, r.take(clock.now(), false, 1.0))
	assert.Equal(t, r.quotaBalance, 7.0)
}

func TestResetQuotaUsageRotation(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000000,
	}

	r := &reservoir{
		quota:    5,
		capacity: 100,
	}

	// consume quota for second
	for i := 0; i < 5; i++ {
		assert.True(t, r.take(clock.now(), false, 1.0))
	}

	// take() should be false since no unused quota left
	taken := r.take(clock.now(), false, 1.0)
	assert.False(t, taken)

	// increment epoch to reset unused quota
	clock = &mockClock{
		nowTime: 1500000001,
	}

	// take() should be true since ununsed quota is available
	assert.True(t, r.take(clock.now(), false, 1.0))
}

// assert that when quotaBalance exceeds totalQuotaBalanceCapacity then totalQuotaBalanceCapacity
// gets assigned to quotaBalance.
func TestQuotaBalanceNonBorrowExceedsCapacity(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000001,
	}

	r := &reservoir{
		quota:    6,
		capacity: 5,
		lastTick: time.Unix(1500000000, 0),
	}

	r.refreshQuotaBalanceLocked(clock.now(), false)

	// assert that if quotaBalance exceeds capacity then total capacity would be new quotaBalance
	assert.Equal(t, r.quotaBalance, r.capacity)
}

// assert quotaBalance and capacity of borrowing case.
func TestQuotaBalanceBorrow(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000001,
	}

	r := &reservoir{
		quota:    6,
		capacity: 5,
		lastTick: time.Unix(1500000000, 0),
	}

	r.refreshQuotaBalanceLocked(clock.now(), true)

	// assert that if quotaBalance exceeds capacity then total capacity would be new quotaBalance
	assert.Equal(t, 1.0, r.quotaBalance)
}

// assert that when borrow is true and elapsedTime is greater than 1, then we only increase the quota balance by 1.
func TestQuotaBalanceIncreaseByOneBorrowCase(t *testing.T) {
	clock := &mockClock{
		nowTime: 1500000002,
	}

	r := &reservoir{
		quota:        6,
		capacity:     5,
		quotaBalance: 0.25,
		lastTick:     time.Unix(1500000000, 0),
	}

	r.refreshQuotaBalanceLocked(clock.now(), true)

	assert.Equal(t, 1.25, r.quotaBalance)
}
