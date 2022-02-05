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

// ToDo: other fields will be used in business logic for remote sampling
// reservoir is a reservoir distributed among all running instances of the SDK
type reservoir struct {
	// Quota assigned to client
	//quota int64
	//
	// Quota refresh timestamp
	//refreshedAt int64
	//
	// Quota expiration timestamp
	//expiresAt int64
	//
	// Polling interval for quota
	interval int64
	//
	// True if reservoir has been borrowed from this epoch
	//borrowed bool

	// Total size of reservoir
	capacity int64
	//
	// Reservoir consumption for current epoch
	//used int64
	//
	// Unix epoch. Reservoir usage is reset every second.
	//currentEpoch int64
}
