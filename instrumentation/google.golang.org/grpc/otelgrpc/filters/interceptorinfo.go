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

package filters // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"

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
type InterceptorInfo struct {
	ctx    context.Context
	method string
	desc   *grpc.StreamDesc
	usinfo *grpc.UnaryServerInfo
	ssinfo *grpc.StreamServerInfo
	typ    interceptorType
}

// NewUnaryClientInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to UnaryClientInterceptor.
func NewUnaryClientInterceptorInfo(
	ctx context.Context,
	method string,
) *InterceptorInfo {
	return &InterceptorInfo{
		ctx:    ctx,
		method: method,
		typ:    unaryClient,
	}
}

// NewStreamClientInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to StreamServerInterceptor.
func NewStreamClientInterceptorInfo(
	ctx context.Context,
	desc *grpc.StreamDesc,
	method string,
) *InterceptorInfo {
	return &InterceptorInfo{
		ctx:    ctx,
		desc:   desc,
		method: method,
		typ:    streamClient,
	}
}

// NewUnaryServerInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to UnaryServerInterceptor.
func NewUnaryServerInterceptorInfo(
	ctx context.Context,
	info *grpc.UnaryServerInfo,
) *InterceptorInfo {
	return &InterceptorInfo{
		ctx:    ctx,
		usinfo: info,
		typ:    unaryServer,
	}
}

// NewStreamServerInterceptorInfo return a pointer of InterceptorInfo
// based on the argument passed to StreamServerInterceptor.
func NewStreamServerInterceptorInfo(
	info *grpc.StreamServerInfo,
) *InterceptorInfo {
	return &InterceptorInfo{
		ssinfo: info,
		typ:    streamServer,
	}
}
