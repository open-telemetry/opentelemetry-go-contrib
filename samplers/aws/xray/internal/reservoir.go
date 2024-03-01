// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"sync"
	"time"
)

// reservoir represents a sampling statistics for a given rule and populate it's value from
// the response getSamplingTargets API which sends information on sampling statistics in real-time.
type reservoir struct {
	// Quota expiration timestamp.
	expiresAt time.Time

	// Quota assigned to client to consume per second.
	quota float64

	// Current balance of quota.
	quotaBalance float64

	// Total size of reservoir consumption per second.
	capacity float64

	// Quota refresh timestamp.
	refreshedAt time.Time

	// Polling interval for quota.
	interval time.Duration

	// Stores reservoir ticks.
	lastTick time.Time

	mu sync.RWMutex
}

// expired returns true if current time is past expiration timestamp. Otherwise, false is returned if no quota remains.
func (r *reservoir) expired(now time.Time) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return now.After(r.expiresAt)
}

// take consumes quota from reservoir, if any remains, then returns true. False otherwise.
func (r *reservoir) take(now time.Time, borrowed bool, itemCost float64) bool { // nolint: revive  // borrowed is not a control flag.
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.capacity == 0 {
		return false
	}

	if r.lastTick.IsZero() {
		r.lastTick = now

		if borrowed {
			r.quotaBalance = 1.0
		} else {
			r.quotaBalance = r.quota
		}
	}

	if r.quotaBalance >= itemCost {
		r.quotaBalance -= itemCost
		return true
	}

	// update quota balance based on elapsed time
	r.refreshQuotaBalanceLocked(now, borrowed)

	if r.quotaBalance >= itemCost {
		r.quotaBalance -= itemCost
		return true
	}

	return false
}

// refreshQuotaBalanceLocked refreshes the quotaBalance. If borrowed is true then add to the quota balance 1 by every second,
// otherwise add to the quota balance based on assigned quota by X-Ray service.
// It is assumed the lock is held when calling this.
func (r *reservoir) refreshQuotaBalanceLocked(now time.Time, borrowed bool) { // nolint: revive  // borrowed is not a control flag.
	elapsedTime := now.Sub(r.lastTick)
	r.lastTick = now

	// Calculate how much credit have we accumulated since the last tick.
	if borrowed {
		// In borrowing case since we want to enforce sample one req every second, no need to accumulate
		// quotaBalance based on elapsedTime when elapsedTime is greater than 1.
		if elapsedTime.Seconds() > 1.0 {
			r.quotaBalance += 1.0
		} else {
			r.quotaBalance += elapsedTime.Seconds()
		}
	} else {
		elapsedSeconds := elapsedTime.Seconds()
		r.quotaBalance += elapsedSeconds * r.quota

		if r.quotaBalance > r.quota {
			r.quotaBalance = r.quota
		}
	}
}
