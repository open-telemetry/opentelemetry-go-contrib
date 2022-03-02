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
	"sync/atomic"
	"time"
)

// reservoir represents a sampling statistics for a given rule and populate it's value from
// the response getSamplingTargets API which sends information on sampling statistics real-time
type reservoir struct {
	// quota expiration timestamp
	expiresAt int64

	// reservoir usage is reset every second
	currentEpoch int64

	// reservoir consumption for current epoch
	used int64

	// quota assigned to client
	quota int64

	// quota refresh timestamp
	refreshedAt time.Time

	// polling interval for quota
	interval time.Duration

	// total size of reservoir
	capacity int64
}

// expired returns true if current time is past expiration timestamp. False otherwise.
func (r *reservoir) expired(now int64) bool {
	expire := atomic.LoadInt64(&r.expiresAt)

	return now >= expire
}

// borrow returns true if the reservoir has not been borrowed from this epoch.
func (r *reservoir) borrow(now int64) bool {
	cur := atomic.LoadInt64(&r.currentEpoch)
	if cur >= now {
		return false
	}
	return atomic.CompareAndSwapInt64(&r.currentEpoch, cur, now)
}

// Take consumes quota from reservoir, if any remains, then returns true. False otherwise.
func (r *reservoir) take(now int64) bool {
	var mu sync.RWMutex
	mu.Lock()
	defer mu.Unlock()

	if r.currentEpoch != now {
		r.used = 0
		r.currentEpoch = now
	}

	if r.quota > r.used {
		r.used++
		return true
	}

	return false
}
