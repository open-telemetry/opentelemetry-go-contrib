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
	"context"
	"time"

	"github.com/benbjohnson/clock"
	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/dynamicconfig/v1"
	resourcepb "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// A ServiceReader periodically reads from a remote configuration service to get configs that apply
// to the SDK.
type ServiceReader struct {
	// Used for testing purposes.
	clock clock.Clock

	// Required
	configHost string

	// Timestamp of last time config service was checked.
	lastTimestamp time.Time

	// Most recent config version.
	lastKnownFingerprint []byte

	// Suggested time from reading from the config service to wait before checking
	// config service again (seconds).
	suggestedWaitTimeSec int32

	// Required. Label to identify this instance.
	resource *resourcepb.Resource
}

func NewServiceReader(configHost string, resource *resourcepb.Resource) *ServiceReader {
	return &ServiceReader{
		clock:      clock.New(),
		configHost: configHost,
		resource:   resource,
	}
}

// Reads from a config service. readConfig() will cause thread to sleep until
// suggestedWaitTimeSec.
func (r *ServiceReader) readConfig() (*Config, error) {
	// suggstedWaitTime is how much longer to wait before reaching the full
	// ServiceReader.suggestedWaitTimeSec.
	suggestedWaitTime := r.suggestedWaitTime()
	time.Sleep(suggestedWaitTime)
	r.suggestedWaitTimeSec = 0

	// Get the new config.
	conn, err := grpc.Dial(r.configHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewDynamicConfigClient(conn)

	request := &pb.ConfigRequest{
		LastKnownFingerprint: r.lastKnownFingerprint,
		Resource:             r.resource,
	}

	md := metadata.Pairs("timestamp", r.clock.Now().Format(time.StampNano))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	response, err := c.GetConfig(ctx, request)
	if err != nil {
		return nil, err
	}

	r.lastKnownFingerprint = response.Fingerprint
	r.lastTimestamp = r.clock.Now()
	r.suggestedWaitTimeSec = response.SuggestedWaitTimeSec

	newConfig := Config{
		pb.ConfigResponse{
			Fingerprint:  response.Fingerprint,
			MetricConfig: response.MetricConfig,
			TraceConfig:  response.TraceConfig,
		},
	}

	return &newConfig, nil
}

// Returns how much longer we need to wait to reach the full suggestedWaitTimeSec.
func (r *ServiceReader) suggestedWaitTime() time.Duration {
	if r.lastTimestamp.IsZero() || r.suggestedWaitTimeSec == 0 {
		return 0
	}

	// This is the suggested earliest time we should read from the config service again.
	suggestedReadTime := r.lastTimestamp.Add(time.Duration(r.suggestedWaitTimeSec) * time.Second)

	suggestedWaitTime := suggestedReadTime.Sub(r.clock.Now())
	if suggestedWaitTime < 0 {
		suggestedWaitTime = 0
	}

	// Return the time needed to wait to reach suggestedReadTime.
	return suggestedWaitTime
}
