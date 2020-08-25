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

import (
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
)

const (
	// cassVersionKey is the key for the attribute/label describing
	// the cql version.
	cassVersionKey = label.Key("db.cassandra.version")

	// cassHostIDKey is the key for the attribute/label describing the id
	// of the host being queried.
	cassHostIDKey = label.Key("db.cassandra.host.id")

	// cassHostStateKey is the key for the attribute/label describing
	// the state of the casssandra server hosting the node being queried.
	cassHostStateKey = label.Key("db.cassandra.host.state")

	// cassBatchQueriesKey is the key for the attribute describing
	// the number of queries contained within the batch statement.
	cassBatchQueriesKey = label.Key("db.cassandra.batch.queries")

	// cassErrMsgKey is the key for the attribute/label describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	cassErrMsgKey = label.Key("db.cassandra.error.message")

	// cassRowsReturnedKey is the key for the span attribute describing the number of rows
	// returned on a query to the database.
	cassRowsReturnedKey = label.Key("db.cassandra.rows.returned")

	// cassQueryAttemptsKey is the key for the span attribute describing the number of attempts
	// made for the query in question.
	cassQueryAttemptsKey = label.Key("db.cassandra.attempts")

	// Static span names
	cassBatchQueryName = "Batch Query"
	cassConnectName    = "New Connection"

	// instrumentationName is the name of the instrumentation package.
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql"
)

// ------------------------------------------ Connection-level Attributes

// cassDBSystem returns the name of the DB system,
// cassandra, as a KeyValue pair (db.system).
func cassDBSystem() label.KeyValue {
	return semconv.DBSystemCassandra
}

// cassPeerName returns the hostname of the cassandra
// server as a semconv KeyValue pair (net.peer.name).
func cassPeerName(name string) label.KeyValue {
	return semconv.NetPeerNameKey.String(name)
}

// cassPeerPort returns the port number of the cassandra
// server as a semconv KeyValue pair (net.peer.port).
func cassPeerPort(port int) label.KeyValue {
	return semconv.NetPeerPortKey.Int(port)
}

// cassPeerIP returns the IP address of the cassandra
// server as a semconv KeyValue pair (net.peer.ip).
func cassPeerIP(ip string) label.KeyValue {
	return semconv.NetPeerIPKey.String(ip)
}

// cassVersion returns the cql version as a KeyValue pair.
func cassVersion(version string) label.KeyValue {
	return cassVersionKey.String(version)
}

// cassHostID returns the id of the cassandra host as a KeyValue pair.
func cassHostID(id string) label.KeyValue {
	return cassHostIDKey.String(id)
}

// cassHostState returns the state of the cassandra host as a KeyValue pair.
func cassHostState(state string) label.KeyValue {
	return cassHostStateKey.String(state)
}

// ------------------------------------------ Call-level attributes

// cassStatement returns the statement made to the cassandra database as a
// semconv KeyValue pair (db.statement).
func cassStatement(stmt string) label.KeyValue {
	return semconv.DBStatementKey.String(stmt)
}

// cassDBOperation returns the batch query operation
// as a semconv KeyValue pair (db.operation). This is used in lieu of a
// db.statement, which is not feasible to include in a span for a batch query
// because there can be n different query statements in a batch query.
func cassBatchQueryOperation() label.KeyValue {
	cassBatchQueryOperation := "db.cassandra.batch.query"
	return semconv.DBOperationKey.String(cassBatchQueryOperation)
}

// cassConnectOperation returns the connect operation
// as a semconv KeyValue pair (db.operation). This is used in lieu of a
// db.statement since connection creation does not have a CQL statement.
func cassConnectOperation() label.KeyValue {
	cassConnectOperation := "db.cassandra.connect"
	return semconv.DBOperationKey.String(cassConnectOperation)
}

// cassKeyspace returns the keyspace of the session as
// a semconv KeyValue pair (db.cassandra.keyspace).
func cassKeyspace(keyspace string) label.KeyValue {
	return semconv.DBCassandraKeyspaceKey.String(keyspace)
}

// cassBatchQueries returns the number of queries in a batch query
// as a KeyValue pair.
func cassBatchQueries(num int) label.KeyValue {
	return cassBatchQueriesKey.Int(num)
}

// cassErrMsg returns the KeyValue pair of an error message
// encountered when executing a query, batch query, or error.
func cassErrMsg(msg string) label.KeyValue {
	return cassErrMsgKey.String(msg)
}

// cassRowsReturned returns the KeyValue pair of the number of rows
// returned from a query.
func cassRowsReturned(rows int) label.KeyValue {
	return cassRowsReturnedKey.Int(rows)
}

// cassQueryAttempts returns the KeyValue pair of the number of attempts
// made for a query.
func cassQueryAttempts(num int) label.KeyValue {
	return cassQueryAttemptsKey.Int(num)
}
