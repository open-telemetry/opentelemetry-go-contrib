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

package internal_xray

type reservoir struct {
	// Quota assigned to client
	quota int64

	// Quota refresh timestamp
	refreshedAt int64

	// Quota expiration timestamp
	expiresAt int64

	// Polling interval for quota
	interval int64

	// Total size of reservoir
	capacity int64

	// Reservoir consumption for current epoch
	used int64

	// Unix epoch. Reservoir usage is reset every second.
	currentEpoch int64
}

// expired returns true if current time is past expiration timestamp. False otherwise.
func (r *reservoir) expired(now int64) bool {
	//expire := atomic.LoadInt64(&r.expiresAt)
	//
	//return now > expire
	return false
}

// borrow returns true if the reservoir has not been borrowed from this epoch
func (r *reservoir) borrow(now int64) bool {
	//cur := atomic.LoadInt64(&r.currentEpoch)
	//if cur >= now {
	//	return false
	//}
	//return atomic.CompareAndSwapInt64(&r.currentEpoch, cur, now)
	return false
}

// Take consumes quota from reservoir, if any remains, and returns true. False otherwise.
func (r *reservoir) take(now int64) bool {
	//cur := atomic.LoadInt64(&r.currentEpoch)
	//quota := atomic.LoadInt64(&r.quota)
	//used := atomic.LoadInt64(&r.used)
	//
	//if cur != now {
	//	atomic.CompareAndSwapInt64(&r.currentEpoch, cur, now)
	//	atomic.CompareAndSwapInt64(&r.used, used, int64(0))
	//	used = 0
	//}
	//
	//if quota > used {
	//	atomic.AddInt64(&r.used, 1)
	//	return true
	//}
	return false
}
