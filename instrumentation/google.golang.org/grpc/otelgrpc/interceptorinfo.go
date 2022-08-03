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
	"path"
	"strings"

	"google.golang.org/grpc"
)

// interceptorType is the flag to define which gRPC interceptor
// the interceptorInfo object is.
type interceptorType uint8

const (
	unaryClient interceptorType = iota
	streamClient
	unaryServer
	streamServer
)

// interceptorInfo is the union of some arguments to four types of
// gRPC interceptors.
type interceptorInfo struct {
	method string
	usinfo *grpc.UnaryServerInfo
	ssinfo *grpc.StreamServerInfo
	typ    interceptorType
}

// splitFullMethod splits path defined in gRPC protocol
// and returns as gRPCPath object that has divided service and method names
// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md
// If name is not FullMethod, returned gRPCPath has empty service field.
func splitFullMethod(name string) gRPCPath {
	s, m := path.Split(name)
	if s != "" {
		s = path.Clean(s)
		s = strings.TrimLeft(s, "/")
	}
	return gRPCPath{
		service: s,
		method:  m,
	}
}

func (i *interceptorInfo) splitFullMethod() gRPCPath {
	var p gRPCPath
	switch i.typ {
	case unaryServer:
		p = splitFullMethod(i.usinfo.FullMethod)
	case streamServer:
		p = splitFullMethod(i.ssinfo.FullMethod)
	case unaryClient, streamClient:
		p = splitFullMethod(i.method)
	default:
		p = gRPCPath{
			method: i.method,
		}
	}
	return p
}

// newUnaryClientInterceptorInfo return a pointer of interceptorInfo
// based on the argument passed to UnaryClientInterceptor.
func newUnaryClientInterceptorInfo(
	method string,
) *interceptorInfo {
	return &interceptorInfo{
		method: method,
		typ:    unaryClient,
	}
}

// newStreamClientInterceptorInfo return a pointer of interceptorInfo
// based on the argument passed to StreamServerInterceptor.
func newStreamClientInterceptorInfo(
	method string,
) *interceptorInfo {
	return &interceptorInfo{
		method: method,
		typ:    streamClient,
	}
}

// newUnaryServerInterceptorInfo return a pointer of interceptorInfo
// based on the argument passed to UnaryServerInterceptor.
func newUnaryServerInterceptorInfo(
	info *grpc.UnaryServerInfo,
) *interceptorInfo {
	return &interceptorInfo{
		usinfo: info,
		typ:    unaryServer,
	}
}

// newStreamServerInterceptorInfo return a pointer of interceptorInfo
// based on the argument passed to StreamServerInterceptor.
func newStreamServerInterceptorInfo(
	info *grpc.StreamServerInfo,
) *interceptorInfo {
	return &interceptorInfo{
		ssinfo: info,
		typ:    streamServer,
	}
}
