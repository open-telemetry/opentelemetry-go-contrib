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

package notify

import (
	"errors"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
)

// A configuration used in the SDK to dynamically change metric collection and tracing.
type MetricConfig struct {
	pb.MetricConfigResponse
}

func (config *MetricConfig) Validate() error {
	for _, schedule := range config.Schedules {
		if schedule.PeriodSec < 0 {
			return errors.New("periods must be positive")
		}
	}

	return nil
}
