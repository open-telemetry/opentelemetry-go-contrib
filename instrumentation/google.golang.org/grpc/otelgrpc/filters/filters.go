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
	"path"
	"strings"
)

// Filter is a predicate used to determine whether a given request in
// interceptor info should be traced. A Filter must return true if
// the request should be traced.
type Filter func(*InterceptorInfo) bool

type gRPCPath struct {
	service string
	method  string
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

// Any takes a list of Filters and returns a Filter that
// returns true if any Filter in the list returns true.
func Any(fs ...Filter) Filter {
	return func(i *InterceptorInfo) bool {
		for _, f := range fs {
			if f(i) {
				return true
			}
		}
		return false
	}
}

// All takes a list of Filters and returns a Filter that
// returns true only if all Filters in the list return true.
func All(fs ...Filter) Filter {
	return func(i *InterceptorInfo) bool {
		for _, f := range fs {
			if !f(i) {
				return false
			}
		}
		return true
	}
}

// None takes a list of Filters and returns a Filter that returns
// true only if none of the Filters in the list return true.
func None(fs ...Filter) Filter {
	return func(i *InterceptorInfo) bool {
		for _, f := range fs {
			if f(i) {
				return false
			}
		}
		return true
	}
}

// Not provides a convenience mechanism for inverting a Filter.
func Not(f Filter) Filter {
	return func(i *InterceptorInfo) bool {
		return !f(i)
	}
}

// MethodName returns a Filter that returns true if the request's
// method name matches the provided string n. This filter is only for
// client interceptors.
func MethodName(n string) Filter {
	return func(i *InterceptorInfo) bool {
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
		return p.method == n
	}
}

// MethodPrefix returns a Filter that returns true if the request's
// method starts with the provided string pre. This filter is only for
// client interceptors.
func MethodPrefix(pre string) Filter {
	return func(i *InterceptorInfo) bool {
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
		return strings.HasPrefix(p.method, pre)
	}
}

// FullMethodName returns a Filter that returns true if the request's
// full RPC method string, i.e. /package.service/method, starts with
// the provided string n. This filter is only for server interceptors.
func FullMethodName(n string) Filter {
	return func(i *InterceptorInfo) bool {
		var fm string
		switch i.typ {
		case unaryClient, streamClient:
			fm = i.method
		case unaryServer:
			fm = i.usinfo.FullMethod
		case streamServer:
			fm = i.ssinfo.FullMethod
		default:
			fm = i.method
		}
		return fm == n
	}
}

// ServiceName returns a Filter that returns true if the request's
// service name, i.e. package.service, matches s.
func ServiceName(s string) Filter {
	return func(i *InterceptorInfo) bool {
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
		return p.service == s
	}
}

// ServicePrefix returns a Filter that returns true if the request's
// service name, i.e. package.service, starts with the provided string pre.
func ServicePrefix(pre string) Filter {
	return func(i *InterceptorInfo) bool {
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
		return strings.HasPrefix(p.service, pre)
	}
}

// NotHealthCheck returns a Filter that returns true if the request's
// is not health check defined by gRPC Health Checking Protocol.
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
// This is the short of Not(MethodName("grpc.health.v1.Health/Check")).
func NotHealthCheck() Filter {
	return Not(ServicePrefix("grpc.health.v1.Health"))
}
