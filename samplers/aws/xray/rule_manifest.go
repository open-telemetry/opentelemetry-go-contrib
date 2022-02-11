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
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal_xray"
	"sort"
	"strings"
)

const defaultInterval = int64(10)

const manifestTTL = 3600 // Seconds

// manifest represents a full sampling ruleset, with a list of
// custom rules and default values for incoming requests that do
// not match any of the provided rules.
type manifest struct {
	rules       []*rule
	index       map[string]*rule
	refreshedAt int64
	clock       internal_xray.clock
}

// createRule creates a user-defined rule, appends it to the sorted array,
// adds it to the index, and returns the newly created rule.
