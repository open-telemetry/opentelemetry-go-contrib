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
	// cassVersionKey is the key for the span attribute describing
	// the cql version.
	cassVersionKey = kv.Key("cassandra.version")

	// cassHostKey is the key for the span attribute describing
	// the host of the cassandra instance being queried.
	cassHostKey = kv.Key("cassandra.host")

	// cassHostIDKey is the key for the metric label describing the id
	// of the host being queried.
	cassHostIDKey = kv.Key("cassandra.host.id")

	// cassPortKey is the key for the span attribute describing
	// the port of the cassandra server being queried.
	cassPortKey = kv.Key("cassandra.port")

	// cassHostStateKey is the key for the span attribute describing
	// the state of the casssandra server hosting the node being queried.
	cassHostStateKey = kv.Key("cassandra.host.state")

	// cassKeyspaceKey is the key for the KeyValue pair describing the
	// keyspace of the current session.
	cassKeyspaceKey = kv.Key("cassandra.keyspace")

	// cassStatementKey is the key for the span attribute describing the
	// the statement used to query the cassandra database.
	// This attribute will only be found on a span for a query.
	cassStatementKey = kv.Key("cassandra.stmt")

	// cassBatchQueriesKey is the key for the span attributed describing
	// the number of queries contained within the batch statement.
	cassBatchQueriesKey = kv.Key("cassandra.batch.queries")

	// cassErrMsgKey is the key for the span attribute describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	cassErrMsgKey = kv.Key("cassandra.err.msg")

	// cassRowsReturnedKey is the key for the span attribute describing the number of rows
	// returned on a query to the database.
	cassRowsReturnedKey = kv.Key("cassandra.rows.returned")

	// cassQueryAttemptsKey is the key for the span attribute describing the number of attempts
	// made for the query in question.
	cassQueryAttemptsKey = kv.Key("cassandra.attempts")

	// cassQueryAttemptNumKey is the key for the span attribute describing
	// which attempt the current query is as a 0-based index.
	cassQueryAttemptNumKey = kv.Key("cassandra.attempt")

	// Names of the spans for query, batch query, and connect respectively.
	cassQueryName      = "cassandra.query"
	cassBatchQueryName = "cassandra.batch.query"
	cassConnectName    = "cassandra.connect"
)

// cassVersion returns the cql version as a KeyValue pair.
func cassVersion(version string) kv.KeyValue {
	return cassVersionKey.String(version)
}

// cassHost returns the cassandra host as a KeyValue pair.
func cassHost(host string) kv.KeyValue {
	return cassHostKey.String(host)
}

// cassHostID returns the id of the cassandra host as a KeyValue pair.
func cassHostID(id string) kv.KeyValue {
	return cassHostIDKey.String(id)
}

// cassPort returns the port of the cassandra node being queried
// as a KeyValue pair.
func cassPort(port int) kv.KeyValue {
	return cassPortKey.Int(port)
}

// cassHostState returns the state of the cassandra host as a KeyValue pair.
func cassHostState(state string) kv.KeyValue {
	return cassHostStateKey.String(state)
}

// cassKeyspace returns the keyspace of the session as a KeyValue pair.
func cassKeyspace(keyspace string) kv.KeyValue {
	return cassKeyspaceKey.String(keyspace)
}

// cassStatement returns the statement made to the cassandra database as a
// KeyValue pair.
func cassStatement(stmt string) kv.KeyValue {
	return cassStatementKey.String(stmt)
}

// cassBatchQueries returns the number of queries in a batch query
// as a KeyValue pair.
func cassBatchQueries(num int) kv.KeyValue {
	return cassBatchQueriesKey.Int(num)
}

// cassErrMsg returns the KeyValue pair of an error message
// encountered when executing a query, batch query, or error.
func cassErrMsg(msg string) kv.KeyValue {
	return cassErrMsgKey.String(msg)
}

// cassRowsReturned returns the KeyValue pair of the number of rows
// returned from a query.
func cassRowsReturned(rows int) kv.KeyValue {
	return cassRowsReturnedKey.Int(rows)
}

// cassQueryAttempts returns the KeyValue pair of the number of attempts
// made for a query.
func cassQueryAttempts(num int) kv.KeyValue {
	return cassQueryAttemptsKey.Int(num)
}

// cassQueryAttemptNum returns the 0-based index attempt number of a
// query as a KeyValue pair.
func cassQueryAttemptNum(num int) kv.KeyValue {
	return cassQueryAttemptNumKey.Int(num)
}
