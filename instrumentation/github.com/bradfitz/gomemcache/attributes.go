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
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
)

const (
	operationAdd            = "add"
	operationCompareAndSwap = "cas"
	operationDecrement      = "decr"
	operationDelete         = "delete"
	operationDeleteAll      = "delete_all"
	operationFlushAll       = "flush_all"
	operationGet            = "get"
	operationIncrement      = "incr"
	operationPing           = "ping"
	operationReplace        = "replace"
	operationSet            = "set"
	operationTouch          = "touch"

	mamcacheDBSystemValue = "memcached"

	memcacheDBItemKeyKeyName = "db.memcached.itemKey"
)

func memcacheDBSystem() kv.KeyValue {
	return standard.DBSystemKey.String(mamcacheDBSystemValue)
}

func memcacheDBOperation(opName string) kv.KeyValue {
	return standard.DBOperationKey.String(opName)
}

func memcacheDBItemKeys(itemKeys ...string) []kv.KeyValue {
	kvs := make([]kv.KeyValue, 0, len(itemKeys))
	for _, ik := range itemKeys {
		kvs = append(kvs, kv.Key(memcacheDBItemKeyKeyName).String(ik))
	}
	return kvs
}
