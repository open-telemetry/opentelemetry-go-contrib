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

	// CassHostIDKey is the key for the metric label describing the id
	// of the host being queried.
	CassHostIDKey = kv.Key("cassandra.host.id")

	// CassPortKey is the key for the span attribute describing
	// the port of the cassandra server being queried.
	CassPortKey = kv.Key("cassandra.port")

	// CassHostStateKey is the key for the span attribute describing
	// the state of the casssandra server hosting the node being queried.
	CassHostStateKey = kv.Key("cassandra.host.state")

	// CassStatementKey is the key for the span attribute describing the
	// the statement used to query the cassandra database.
	// This attribute will only be found on a span for a query.
	CassStatementKey = kv.Key("cassandra.stmt")

	// CassBatchQueriesKey is the key for the span attributed describing
	// the number of queries contained within the batch statement.
	CassBatchQueriesKey = kv.Key("cassandra.batch.queries")

	// CassErrMsgKey is the key for the span attribute describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	CassErrMsgKey = kv.Key("cassandra.err.msg")

	// CassRowsReturnedKey is the key for the span attribute describing the number of rows
	// returned on a query to the database.
	CassRowsReturnedKey = kv.Key("cassandra.rows.returned")

	// CassQueryAttemptsKey is the key for the span attribute describing the number of attempts
	// made for the query in question.
	CassQueryAttemptsKey = kv.Key("cassandra.attempts")

	// CassQueryAttemptNumKey is the key for the span attribute describing
	// which attempt the current query is as a 0-based index.
	CassQueryAttemptNumKey = kv.Key("cassandra.attempt")

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

// CassHostID returns the id of the cassandra host as a keyvalue pair.
func CassHostID(id string) kv.KeyValue {
	return CassHostIDKey.String(id)
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

// CassBatchQueries returns the number of queries in a batch query
// as a keyvalue pair.
func CassBatchQueries(num int) kv.KeyValue {
	return CassBatchQueriesKey.Int(num)
}

// CassErrMsg returns the keyvalue pair of an error message
// encountered when executing a query, batch query, or error.
func CassErrMsg(msg string) kv.KeyValue {
	return CassErrMsgKey.String(msg)
}

// CassRowsReturned returns the keyvalue pair of the number of rows
// returned from a query.
func CassRowsReturned(rows int) kv.KeyValue {
	return CassRowsReturnedKey.Int(rows)
}

// CassQueryAttempts returns the keyvalue pair of the number of attempts
// made for a query.
func CassQueryAttempts(num int) kv.KeyValue {
	return CassQueryAttemptsKey.Int(num)
}

// CassQueryAttemptNum returns the 0-based index attempt number of a
// query as a keyvalue pair.
func CassQueryAttemptNum(num int) kv.KeyValue {
	return CassQueryAttemptNumKey.Int(num)
}
