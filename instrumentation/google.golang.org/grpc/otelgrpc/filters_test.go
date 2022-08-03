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
	"testing"

	"google.golang.org/grpc"
)

type testCase struct {
	name string
	i    *interceptorInfo
	f    Filter
	want bool
}

func dummyUnaryServerInfo(n string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{
		FullMethod: n,
	}
}

func dummyStreamServerInfo(n string) *grpc.StreamServerInfo {
	return &grpc.StreamServerInfo{
		FullMethod: n,
	}
}

func TestMethodName(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/Hello"
	tcs := []testCase{
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    MethodName("Hello"),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethodName),
			f:    MethodName("Hello"),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    MethodName("Hello"),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethodName)),
			f:    MethodName("Hello"),
			want: true,
		},
		{
			name: "unary client interceptor fail",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    MethodName("Goodbye"),
			want: false,
		},
	}

	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestMethodPrefix(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    MethodPrefix("Foobar"),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethodName),
			f:    MethodPrefix("Foobar"),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    MethodPrefix("Foobar"),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethodName)),
			f:    MethodPrefix("Foobar"),
			want: true,
		},
		{
			name: "unary client interceptor fail",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    MethodPrefix("Barfoo"),
			want: false,
		},
	}
	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestFullMethodName(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/Hello"
	tcs := []testCase{
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    FullMethodName(dummyFullMethodName),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethodName),
			f:    FullMethodName(dummyFullMethodName),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    FullMethodName(dummyFullMethodName),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethodName)),
			f:    FullMethodName(dummyFullMethodName),
			want: true,
		},
		{
			name: "unary client interceptor fail",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    FullMethodName("/example.HelloService/Goodbye"),
			want: false,
		},
	}

	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestServiceName(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/Hello"

	tcs := []testCase{
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    ServiceName("example.HelloService"),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethodName),
			f:    ServiceName("example.HelloService"),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    ServiceName("example.HelloService"),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethodName)),
			f:    ServiceName("example.HelloService"),
			want: true,
		},
		{
			name: "unary client interceptor fail",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    ServiceName("opentelemetry.HelloService"),
			want: false,
		},
	}

	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestServicePrefix(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    ServicePrefix("example"),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethodName),
			f:    ServicePrefix("example"),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    ServicePrefix("example"),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethodName)),
			f:    ServicePrefix("example"),
			want: true,
		},
		{
			name: "unary client interceptor fail",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    ServicePrefix("opentelemetry"),
			want: false,
		},
	}
	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestAny(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "unary client interceptor true && true",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    Any(MethodName("FoobarHello"), MethodPrefix("Foobar")),
			want: true,
		},
		{
			name: "unary client interceptor false && true",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    Any(MethodName("Hello"), MethodPrefix("Foobar")),
			want: true,
		},
		{
			name: "unary client interceptor false && false",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    Any(MethodName("Goodbye"), MethodPrefix("Barfoo")),
			want: false,
		},
	}
	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestAll(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "unary client interceptor true && true",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    All(MethodName("FoobarHello"), MethodPrefix("Foobar")),
			want: true,
		},
		{
			name: "unary client interceptor true && false",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    All(MethodName("FoobarHello"), MethodPrefix("Barfoo")),
			want: false,
		},
		{
			name: "unary client interceptor false && false",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    All(MethodName("FoobarGoodbye"), MethodPrefix("Barfoo")),
			want: false,
		},
	}
	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestNone(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "unary client interceptor true && true",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    None(MethodName("FoobarHello"), MethodPrefix("Foobar")),
			want: false,
		},
		{
			name: "unary client interceptor true && false",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    None(MethodName("FoobarHello"), MethodPrefix("Barfoo")),
			want: false,
		},
		{
			name: "unary client interceptor false && false",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    None(MethodName("FoobarGoodbye"), MethodPrefix("Barfoo")),
			want: true,
		},
	}
	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestNot(t *testing.T) {
	const dummyFullMethodName = "/example.HelloService/FoobarHello"
	tcs := []testCase{
		{
			name: "methodname not",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    Not(MethodName("FoobarHello")),
			want: false,
		},
		{
			name: "method prefix not",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethodName)),
			f:    Not(MethodPrefix("FoobarHello")),
			want: false,
		},
		{
			name: "unary client interceptor not all(true && true)",
			i:    newUnaryClientInterceptorInfo(dummyFullMethodName),
			f:    Not(All(MethodName("FoobarHello"), MethodPrefix("Foobar"))),
			want: false,
		},
	}

	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}

func TestNotHealthCheck(t *testing.T) {
	const (
		healthCheck     = "/grpc.health.v1.Health/Check"
		dummyFullMethod = "/example.HelloService/FoobarHello"
	)
	tcs := []testCase{
		{
			name: "unary client interceptor healthcheck",
			i:    newUnaryClientInterceptorInfo(healthCheck),
			f:    NotHealthCheck(),
			want: false,
		},
		{
			name: "stream client interceptor healthcheck",
			i:    newStreamClientInterceptorInfo(healthCheck),
			f:    NotHealthCheck(),
			want: false,
		},
		{
			name: "unary server interceptor healthcheck",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(healthCheck)),
			f:    NotHealthCheck(),
			want: false,
		},
		{
			name: "stream server interceptor healthcheck",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(healthCheck)),
			f:    NotHealthCheck(),
			want: false,
		},
		{
			name: "unary client interceptor",
			i:    newUnaryClientInterceptorInfo(dummyFullMethod),
			f:    NotHealthCheck(),
			want: true,
		},
		{
			name: "stream client interceptor",
			i:    newStreamClientInterceptorInfo(dummyFullMethod),
			f:    NotHealthCheck(),
			want: true,
		},
		{
			name: "unary server interceptor",
			i:    newUnaryServerInterceptorInfo(dummyUnaryServerInfo(dummyFullMethod)),
			f:    NotHealthCheck(),
			want: true,
		},
		{
			name: "stream server interceptor",
			i:    newStreamServerInterceptorInfo(dummyStreamServerInfo(dummyFullMethod)),
			f:    NotHealthCheck(),
			want: true,
		},
	}

	for _, tc := range tcs {
		out := tc.f(tc.i)
		if tc.want != out {
			t.Errorf("test case '%v' failed, wanted %v but obtained %v", tc.name, tc.want, out)
		}
	}
}
