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

package internal // import "go.opentelemetry.io/contrib/instrumentation/github.com/bradfitz/gomemcache/memcache/otelmemcache/internal"

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

type Operation string

// Instrumentation specific tracing information.
const (
	OperationAdd            Operation = "add"
	OperationCompareAndSwap Operation = "cas"
	OperationDecrement      Operation = "decr"
	OperationDelete         Operation = "delete"
	OperationDeleteAll      Operation = "delete_all"
	OperationFlushAll       Operation = "flush_all"
	OperationGet            Operation = "get"
	OperationIncrement      Operation = "incr"
	OperationPing           Operation = "ping"
	OperationReplace        Operation = "replace"
	OperationSet            Operation = "set"
	OperationTouch          Operation = "touch"

	MamcacheDBSystemValue = "memcached"

	MemcacheDBItemKeyName attribute.Key = "db.memcached.item"
)

func MemcacheDBSystem() attribute.KeyValue {
	return semconv.DBSystemKey.String(MamcacheDBSystemValue)
}

func MemcacheDBOperation(opName Operation) attribute.KeyValue {
	return semconv.DBOperationKey.String(string(opName))
}

func MemcacheDBItemKeys(itemKeys ...string) attribute.KeyValue {
	if len(itemKeys) > 1 {
		return MemcacheDBItemKeyName.StringSlice(itemKeys)
	}

	return MemcacheDBItemKeyName.String(itemKeys[0])
}
