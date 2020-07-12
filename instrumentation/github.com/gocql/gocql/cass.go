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

package gocql

import "go.opentelemetry.io/otel/api/kv"

const (
	// CassVersionKey is the key for the span attribute describing
	// the cql version.
	CassVersionKey = kv.Key("cassandra.version")

	// CassHostKey is the key for the span attribute describing
	// the host of the cassandra instance being queried.
	CassHostKey = kv.Key("cassandra.host")

	// CassPortKey is the key for the span attribute describing
	// the port of the cassandra server being queried.
	CassPortKey = kv.Key("cassandra.port")

	// CassHostStateKey is the key for the span attribute describing
	// the state of the casssandra server hosting the node being queried.
	CassHostStateKey = kv.Key("cassandra.host_state")

	// CassStatementKey is the key for the span attribute describing the
	// the statement used to query the cassandra database.
	// This attribute will only be found on a span for a query.
	CassStatementKey = kv.Key("cassandra.stmt")

	// CassBatchStatementsKey is the key for the span attribute describing
	// the list of statments used to query the cassandra database in a batch query.
	// This attribute will only be found on a span for a batch query.
	CassBatchStatementsKey = kv.Key("cassandra.batch_stmts")

	// CassErrMsgKey is the key for the span attribute describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	CassErrMsgKey = kv.Key("cassandra.err_msg")

	// Names of the spans for query, batch query, and connect respectively.
	cassQueryName      = "cassandra.query"
	cassBatchQueryName = "cassandra.batch_query"
	cassConnectName    = "cassandra.connect"
)

// CassVersion returns the cql version as a KeyValue pair.
func CassVersion(version string) kv.KeyValue {
	return CassVersionKey.String(version)
}

// CassHost returns the cassandra host as a KeyValue pair.
func CassHost(host string) kv.KeyValue {
	return CassHostKey.String(host)
}

// CassPort returns the port of the cassandra node being queried
// as a KeyValue pair.
func CassPort(port int) kv.KeyValue {
	return CassPortKey.Int(port)
}

// CassHostState returns the state of the cassandra host as a KeyValue pair.
func CassHostState(state string) kv.KeyValue {
	return CassHostStateKey.String(state)
}

// CassStatement returns the statement made to the cassandra database as a
// KeyValue pair.
func CassStatement(stmt string) kv.KeyValue {
	return CassStatementKey.String(stmt)
}

// CassBatchStatements returns the array of statments executed in a batch
// query made to the cassandra database as a keyvalue pair.
func CassBatchStatements(stmt []string) kv.KeyValue {
	return CassBatchStatementsKey.Array(stmt)
}

// CassErrMsg returns the keyvalue pair of an error message
// encountered when executing a query, batch query, or error.
func CassErrMsg(msg string) kv.KeyValue {
	return CassErrMsgKey.String(msg)
}
