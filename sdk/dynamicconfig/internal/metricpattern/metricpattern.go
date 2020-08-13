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

// Package metricpatern implements the pattern matching "language" for selecting
// metric names.
package metricpattern

import (
	"strings"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
)

// Matches determines whether a name falls in the set of names prescribed by
// the patterns
func Matches(name string, patterns []*pb.MetricConfigResponse_Schedule_Pattern) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		switch m := pattern.Match.(type) {
		case *pb.MetricConfigResponse_Schedule_Pattern_Equals:
			if m.Equals == name {
				return true
			}
		case *pb.MetricConfigResponse_Schedule_Pattern_StartsWith:
			if m.StartsWith == "*" || strings.HasPrefix(name, m.StartsWith) {
				return true
			}
		}
	}

	return false
}
