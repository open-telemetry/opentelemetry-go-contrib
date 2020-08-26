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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

type operation string

const (
	operationAdd            operation = "add"
	operationCompareAndSwap operation = "cas"
	operationDecrement      operation = "decr"
	operationDelete         operation = "delete"
	operationDeleteAll      operation = "delete_all"
	operationFlushAll       operation = "flush_all"
	operationGet            operation = "get"
	operationIncrement      operation = "incr"
	operationPing           operation = "ping"
	operationReplace        operation = "replace"
	operationSet            operation = "set"
	operationTouch          operation = "touch"

	mamcacheDBSystemValue = "memcached"

	memcacheDBItemKeyName label.Key = "db.memcached.item"
)

func memcacheDBSystem() label.KeyValue {
	return semconv.DBSystemKey.String(mamcacheDBSystemValue)
}

func memcacheDBOperation(opName operation) label.KeyValue {
	return semconv.DBOperationKey.String(string(opName))
}

func memcacheDBItemKeys(itemKeys ...string) label.KeyValue {
	if len(itemKeys) > 1 {
		return memcacheDBItemKeyName.Array(itemKeys)
	}

	return memcacheDBItemKeyName.String(itemKeys[0])
}
