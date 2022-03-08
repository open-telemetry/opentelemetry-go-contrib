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

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"sync"
	"time"
)

// reservoir represents a sampling statistics for a given rule and populate it's value from
// the response getSamplingTargets API which sends information on sampling statistics real-time
type reservoir struct {
	// quota expiration timestamp
	expiresAt time.Time

	// quota assigned to client to consume per second
	quota float64

	// current balance of quota
	quotaBalance float64

	// total size of reservoir consumption per second
	capacity float64

	// quota refresh timestamp
	refreshedAt time.Time

	// polling interval for quota
	interval time.Duration

	// stores reservoir ticks
	lastTick time.Time

	// stores borrow ticks
	borrowTick time.Time

	mu *sync.RWMutex
}

// expired returns true if current time is past expiration timestamp. False otherwise.
func (r *reservoir) expired(now time.Time) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return now.After(r.expiresAt)
}

// borrow returns true if the reservoir has not been borrowed from this epoch.
func (r *reservoir) borrow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentTime := now

	if currentTime.Equal(r.borrowTick) {
		return false
	}

	if currentTime.After(r.borrowTick) {
		r.borrowTick = currentTime
		return true
	}

	return false
}

// take consumes quota from reservoir, if any remains, then returns true. False otherwise.
func (r *reservoir) take(now time.Time, itemCost float64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.lastTick.IsZero() {
		r.lastTick = now
		r.quotaBalance = r.quota
	}

	if r.quotaBalance >= itemCost {
		r.quotaBalance -= itemCost
		return true
	}

	// update quota balance based on elapsed time
	r.refreshQuotaBalance(now)

	if r.quotaBalance >= itemCost {
		r.quotaBalance -= itemCost
		return true
	}

	return false
}

func (r *reservoir) refreshQuotaBalance(now time.Time) {
	currentTime := now
	elapsedTime := currentTime.Sub(r.lastTick)
	r.lastTick = currentTime

	// calculate how much credit have we accumulated since the last tick
	totalQuotaBalanceCapacity := elapsedTime.Seconds() * r.capacity
	r.quotaBalance += elapsedTime.Seconds() * r.quota
	if r.quotaBalance > totalQuotaBalanceCapacity {
		r.quotaBalance = totalQuotaBalanceCapacity
	}
}
