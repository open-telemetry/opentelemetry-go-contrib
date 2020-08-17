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
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
)

type MockServer struct {
	pb.UnimplementedMetricConfigServer
	Config *pb.MetricConfigResponse
}

// GetMetricConfig implemented MetricConfigServer
func (server *MockServer) GetMetricConfig(ctx context.Context, in *pb.MetricConfigRequest) (*pb.MetricConfigResponse, error) {
	return server.Config, nil
}

// This function runs a mock config service at an address, serving a defined config.
// It returns a callback that stops the service.
func (server *MockServer) Run(t *testing.T) (func(), string) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get an address: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterMetricConfigServer(srv, server)

	go func() {
		_ = srv.Serve(ln)
	}()

	return func() {
		srv.Stop()
		_ = ln.Close()
	}, ln.Addr().String()
}
