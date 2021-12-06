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

import "sync"

// ToDo: other fields will be used in business logic for remote sampling
// centralizedRule represents a centralized sampling rule
type centralizedRule struct {
	// Centralized reservoir for keeping track of reservoir usage
	reservoir *centralizedReservoir

	// sampling rule properties
	ruleProperties *ruleProperties

	// Number of requests matched against this rule
	//requests float64
	//
	// Number of requests sampled using this rule
	//sampled float64
	//
	// Number of requests burrowed
	//borrows float64
	//
	// Timestamp for last match against this rule
	//usedAt int64

	// Provides system time
	clock Clock

	// Provides random numbers
	rand Rand

	mu sync.RWMutex
}

// properties is the base set of properties that define a sampling rule.
type ruleProperties struct {
	ruleName      string
	serviceType   string
	resourceARN   string
	attributes    map[string]interface{}
	serviceName   string
	host          string
	httpMethod    string
	urlPath       string
	reservoirSize int64
	fixedRate     float64
	priority      int64
	version       int64
}
