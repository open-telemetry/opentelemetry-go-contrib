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

package otelgrpc // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

import (
	"context"

	"google.golang.org/grpc"
)

// interceptorType is the flag to define which gRPC interceptor
// the InterceptorInfo object is.
type interceptorType uint8

const (
	unaryClient interceptorType = iota
	streamClient
	unaryServer
	streamServer
)

// InterceptorInfo is the union of all arguments to four types of
// gRPC interceptors, except for function types and function arguments
// of invoker and streamer:
// * invoker  grpc.UnaryInvoker (UnaryClient)
// * streamer grpc.Streamer (StreamClient)
// * stream   grpc.ServerStream (StreamServer)
// * handler  grpc.UnaryHandler | grpc.StreamHandler (UnaryServer, StreamServer)
// * req, reply, srv interface{} (UnaryClient, UnaryServer, StreamClient)
// * cc       *grpc.ClientConn.
type interceptorInfo struct {
	ctx    context.Context
	method string
	desc   *grpc.StreamDesc
	usinfo *grpc.UnaryServerInfo
	ssinfo *grpc.StreamServerInfo
	typ    interceptorType
}

// newUnaryClientInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to UnaryClientInterceptor.
func newUnaryClientInterceptorInfo(
	ctx context.Context,
	method string,
) *interceptorInfo {
	return &interceptorInfo{
		ctx:    ctx,
		method: method,
		typ:    unaryClient,
	}
}

// newStreamClientInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to StreamServerInterceptor.
func newStreamClientInterceptorInfo(
	ctx context.Context,
	desc *grpc.StreamDesc,
	method string,
) *interceptorInfo {
	return &interceptorInfo{
		ctx:    ctx,
		desc:   desc,
		method: method,
		typ:    streamClient,
	}
}

// newUnaryServerInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to UnaryServerInterceptor.
func newUnaryServerInterceptorInfo(
	ctx context.Context,
	info *grpc.UnaryServerInfo,
) *interceptorInfo {
	return &interceptorInfo{
		ctx:    ctx,
		usinfo: info,
		typ:    unaryServer,
	}
}

// newStreamServerInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to StreamServerInterceptor.
func newStreamServerInterceptorInfo(
	info *grpc.StreamServerInfo,
) *interceptorInfo {
	return &interceptorInfo{
		ssinfo: info,
		typ:    streamServer,
	}
}
