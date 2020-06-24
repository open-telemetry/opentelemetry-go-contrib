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

package dynamicconfig

import (
	"bytes"
	"errors"

	pb "github.com/vmingchen/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"
)

// A configuration used in the SDK to dynamically change metric collection and tracing.
type Config struct {
	pb.ConfigResponse
}

// TODO: Either get rid of this or replace later.
// This is for convenient development/testing purposes.
func GetDefaultConfig(period pb.ConfigResponse_MetricConfig_Schedule_CollectionPeriod, fingerprint []byte) *Config {
	schedule := pb.ConfigResponse_MetricConfig_Schedule{Period: period}

	return &Config{
		pb.ConfigResponse{
			Fingerprint: fingerprint,
			MetricConfig: &pb.ConfigResponse_MetricConfig{
				Schedules: []*pb.ConfigResponse_MetricConfig_Schedule{&schedule},
			},
		},
	}
}

func (config *Config) Validate() error {
	if len(config.MetricConfig.Schedules) != 1 {
		return errors.New("Config must have exactly one Schedule")
	}

	if config.MetricConfig.Schedules[0].Period <= 0 {
		return errors.New("Period must be positive")
	}

	return nil
}

func (config *Config) Equals(otherConfig *Config) bool {
	return bytes.Equal(config.Fingerprint, otherConfig.Fingerprint)
}
