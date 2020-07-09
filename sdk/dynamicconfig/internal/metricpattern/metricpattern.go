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

package metricpattern

import (
	"strings"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"
)

func Matches(name string, patterns []*pb.ConfigResponse_MetricConfig_Schedule_Pattern) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		switch m := pattern.Match.(type) {
		case *pb.ConfigResponse_MetricConfig_Schedule_Pattern_Equals:
			if name == m.Equals {
				return true
			} 
		case *pb.ConfigResponse_MetricConfig_Schedule_Pattern_StartsWith:
			if strings.HasPrefix(name, m.StartsWith) {
				return true
			}
		}
	}

	return false
}
