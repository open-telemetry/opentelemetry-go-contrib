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

package notifier

import (
	"bytes"
	"errors"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
)

// A configuration used in the SDK to dynamically change metric collection and tracing.
type MetricConfig struct {
	pb.MetricConfigResponse
}

// This is for convenient development/testing purposes.
// It produces a Config with a schedule that matches all instruments, with a
// collection period of `period`
func GetDefaultConfig(period int32, fingerprint []byte) *MetricConfig {
	pattern := pb.MetricConfigResponse_Schedule_Pattern{
		Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{StartsWith: "*"},
	}
	schedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{&pattern},
		PeriodSec:         period,
	}

	return &MetricConfig{
		pb.MetricConfigResponse{
			Fingerprint: fingerprint,
			Schedules:   []*pb.MetricConfigResponse_Schedule{&schedule},
		},
	}
}

func (config *MetricConfig) ValidateMetricConfig() error {
	for _, schedule := range config.Schedules {
		if schedule.PeriodSec < 0 {
			return errors.New("Periods must be positive")
		}
	}

	return nil
}

func (config *MetricConfig) Equals(otherConfig *MetricConfig) bool {
	return bytes.Equal(config.Fingerprint, otherConfig.Fingerprint)
}
