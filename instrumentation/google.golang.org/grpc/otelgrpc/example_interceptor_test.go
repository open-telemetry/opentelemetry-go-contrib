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

package otelgrpc_test

import (
	"google.golang.org/grpc"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func ExampleStreamClientInterceptor() {
	_, _ = grpc.Dial("localhost", grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()))
}

func ExampleUnaryClientInterceptor() {
	_, _ = grpc.Dial("localhost", grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()))
}

func ExampleStreamServerInterceptor() {
	_ = grpc.NewServer(grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))
}

func ExampleUnaryServerInterceptor() {
	_ = grpc.NewServer(grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()))
}
