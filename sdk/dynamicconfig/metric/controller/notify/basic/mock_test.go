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

	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

type mockServer struct {
	pb.UnimplementedMetricConfigServer
	config *MetricConfig
}

// GetMetricConfig implemented MetricConfigServer
func (s *mockServer) GetMetricConfig(ctx context.Context, in *pb.MetricConfigRequest) (*pb.MetricConfigResponse, error) {
	return &s.config.MetricConfigResponse, nil
}

// This function runs a mock config service at an address, serving a defined config.
// It returns a callback that stops the service.
func RunMockConfigService(t *testing.T, addr string, config *MetricConfig) func() {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to get an address: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterMetricConfigServer(srv, &mockServer{config: config})

	go func() {
		_ = srv.Serve(ln)
	}()

	return func() {
		srv.Stop()
		_ = ln.Close()
	}
}

func MockResource(serviceName string) *resource.Resource {
	return resource.New(kv.Key(conventions.AttributeServiceName).String(serviceName))
}
