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
// Package main implements a server for Greeter service.

package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/open-telemetry/opentelemetry-proto/gen/go/experimental/metricconfigservice"
)

const (
	port = ":7777"
)

// server is used to implement pb.MetricConfigServer
type server struct {
	pb.UnimplementedMetricConfigServer
}

// GetMetricConfig implemented MetricConfigServer
func (s *server) GetMetricConfig(ctx context.Context, in *pb.MetricConfigRequest) (*pb.MetricConfigResponse, error) {
	log.Printf("Config being read\n")

	pattern1 := pb.MetricConfigResponse_Schedule_Pattern{
		Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{StartsWith: "One"},
	}
	schedule1 := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{&pattern1},
		PeriodSec:         1,
	}
	pattern2 := pb.MetricConfigResponse_Schedule_Pattern{
		Match: &pb.MetricConfigResponse_Schedule_Pattern_Equals{Equals: "Two Metric"},
	}
	schedule2 := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{&pattern2},
		PeriodSec:         5,
	}

	return &pb.MetricConfigResponse{
		Fingerprint: []byte{'b', 'a', 'r'},
		Schedules:   []*pb.MetricConfigResponse_Schedule{&schedule1, &schedule2},
	}, nil
}

func main() {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterMetricConfigServer(s, &server{})
	if err := s.Serve(ln); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
