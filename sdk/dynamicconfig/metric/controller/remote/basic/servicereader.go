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

package basic

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
	resourcepb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/resource/v1"
)

// A ServiceReader periodically reads from a remote configuration service to get
// configs that apply to the SDK.
type ServiceReader struct {
	conn   *grpc.ClientConn
	client pb.MetricConfigClient

	lastKnownFingerprint []byte
	resource             *resourcepb.Resource
}

// NewServiceReader forges a connection with the config service at the address
// in configHost. Additionally it associates the provided resource with all
// communications to the service.
func NewServiceReader(configHost string, resource *resourcepb.Resource) (*ServiceReader, error) {
	conn, err := grpc.Dial(configHost, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("fail to connect to config backend: %w", err)
	}

	return &ServiceReader{
		conn:     conn,
		client:   pb.NewMetricConfigClient(conn),
		resource: resource,
	}, nil
}

// ReadConfig reads and validates the latest configuration data from the
// backend. Returns a nil *MetricConfig if there have been no changes to the
// configuration since the last check.
func (r *ServiceReader) ReadConfig() (*pb.MetricConfigResponse, error) {
	request := &pb.MetricConfigRequest{
		LastKnownFingerprint: r.lastKnownFingerprint,
		Resource:             r.resource,
	}

	response, err := r.client.GetMetricConfig(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("fail to get metric config: %w", err)
	}

	// TODO: SuggestedWaitTimeSec may not be read unless there is a change
	// reflected in the fingerprints
	if r.lastKnownFingerprint != nil && bytes.Equal(r.lastKnownFingerprint, response.Fingerprint) {
		return nil, nil
	}

	r.lastKnownFingerprint = response.Fingerprint

	if err := validate(response); err != nil {
		return nil, fmt.Errorf("metric config invalid: %w", err)
	}

	return response, nil
}

func validate(resp *pb.MetricConfigResponse) error {
	for _, schedule := range resp.Schedules {
		if schedule.PeriodSec < 0 {
			return errors.New("periods must be nonnegative")
		}
	}

	return nil
}

// Stop closes the connection to the config service.
func (r *ServiceReader) Stop() error {
	if err := r.conn.Close(); err != nil {
		return fmt.Errorf("fail to close connection to config backend: %w", err)
	}

	return nil
}
