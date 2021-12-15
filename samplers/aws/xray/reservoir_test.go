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

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTakeQuotaAvailable(t *testing.T) {
	capacity := int64(100)
	used := int64(0)
	quota := int64(9)

	clock := MockClock{
		NowTime: 1500000000,
	}

	r := &centralizedReservoir{
		quota: quota,
		capacity:     capacity,
		used:         used,
		currentEpoch: clock.Now().Unix(),
	}

	s := r.Take(clock.Now().Unix())
	assert.Equal(t, true, s)
	assert.Equal(t, int64(1), r.used)
}

func TestTakeQuotaUnavailable(t *testing.T) {
	capacity := int64(100)
	used := int64(100)
	quota := int64(9)

	clock := MockClock{
		NowTime: 1500000000,
	}

	r := &centralizedReservoir{
		quota: quota,
		capacity:     capacity,
		used:         used,
		currentEpoch: clock.Now().Unix(),
	}

	s := r.Take(clock.Now().Unix())
	assert.Equal(t, false, s)
	assert.Equal(t, int64(100), r.used)
}

func TestExpiredReservoir(t *testing.T) {
	clock := MockClock{
		NowTime: 1500000001,
	}

	r := &centralizedReservoir{
		expiresAt: 1500000000,
	}

	expired := r.expired(clock.Now().Unix())

	assert.Equal(t, true, expired)
}

// Assert that the borrow flag is reset every second
func TestBorrowFlagReset(t *testing.T) {
	clock := MockClock{
		NowTime: 1500000000,
	}

	r := &centralizedReservoir{
		capacity: 10,
	}

	s := r.borrow(clock.Now().Unix())
	assert.True(t, s)

	s = r.borrow(clock.Now().Unix())
	assert.False(t, s)

	// Increment clock by 1
	clock = MockClock{
		NowTime: 1500000001,
	}

	// Reset borrow flag
	r.Take(clock.Now().Unix())

	s = r.borrow(clock.Now().Unix())
	assert.True(t, s)
}

// Assert that the reservoir does not allow borrowing if the reservoir capacity
// is zero.
func TestBorrowZeroCapacity(t *testing.T) {
	clock := MockClock{
		NowTime: 1500000000,
	}

	r := &centralizedReservoir{
		capacity: 0,
	}

	s := r.borrow(clock.Now().Unix())
	assert.False(t, s)
}

func TestResetQuotaUsageRotation(t *testing.T) {
	capacity := int64(100)
	used := int64(0)
	quota := int64(5)

	clock := MockClock{
		NowTime: 1500000000,
	}

	r := &centralizedReservoir{
		quota: quota,
		capacity:     capacity,
		used:         used,
		currentEpoch: clock.Now().Unix(),
	}

	// Consume quota for second
	for i := 0; i < 5; i++ {
		taken := r.Take(clock.Now().Unix())
		assert.Equal(t, true, taken)
		assert.Equal(t, int64(i+1), r.used)
	}

	// Take() should be false since no unused quota left
	taken := r.Take(clock.Now().Unix())
	assert.Equal(t, false, taken)
	assert.Equal(t, int64(5), r.used)

	// Increment epoch to reset unused quota
	clock = MockClock{
		NowTime: 1500000001,
	}

	// Take() should be true since ununsed quota is available
	taken = r.Take(clock.Now().Unix())
	assert.Equal(t, int64(1500000001), r.currentEpoch)
	assert.Equal(t, true, taken)
	assert.Equal(t, int64(1), r.used)
}
