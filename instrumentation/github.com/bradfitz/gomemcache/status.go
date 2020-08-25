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

package gomemcache

import (
	"github.com/bradfitz/gomemcache/memcache"

	"go.opentelemetry.io/otel/codes"
)

// maps memcache error to appropriate error code; otherwise returns status OK
func memcacheErrToStatusCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	switch err {
	case memcache.ErrCacheMiss, memcache.ErrNotStored, memcache.ErrNoStats:
		return codes.NotFound
	case memcache.ErrCASConflict:
		return codes.AlreadyExists
	case memcache.ErrServerError:
		return codes.Internal
	case memcache.ErrMalformedKey:
		return codes.InvalidArgument
	default:
		return codes.Unknown
	}
}
