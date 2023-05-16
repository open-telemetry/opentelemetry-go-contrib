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

package otelttrpc // import "go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc"

import (
	"github.com/containerd/ttrpc"
)

// InterceptorType is the flag to define which ttRPC interceptor
// the InterceptorInfo object is.
type InterceptorType uint8

const (
	// UndefinedInterceptor is the type for the interceptor information that is not
	// well initialized or categorized to other types.
	UndefinedInterceptor InterceptorType = iota
	// UnaryClient is the type for ttrpc.UnaryClient interceptor.
	UnaryClient
	// StreamClient is the type for ttrpc.StreamClient interceptor.
	StreamClient
	// UnaryServer is the type for ttrpc.UnaryServer interceptor.
	UnaryServer
	// StreamServer is the type for ttrpc.StreamServer interceptor.
	StreamServer
)

// InterceptorInfo is the union of some arguments to four types of
// ttRPC interceptors.
type InterceptorInfo struct {
	// Method is method name registered to UnaryClient and StreamClient
	Method string
	// UnaryServerInfo is the metadata for UnaryServer
	UnaryServerInfo *ttrpc.UnaryServerInfo
	// StreamServerInfo if the metadata for StreamServer
	StreamServerInfo *ttrpc.StreamServerInfo
	// Type is the type for interceptor
	Type InterceptorType
}
