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
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
)

const (
	// cassVersionKey is the key for the attribute/label describing
	// the cql version.
	cassVersionKey = kv.Key("db.cassandra.version")

	// cassHostIDKey is the key for the attribute/label describing the id
	// of the host being queried.
	cassHostIDKey = kv.Key("db.cassandra.host.id")

	// cassHostStateKey is the key for the attribute/label describing
	// the state of the casssandra server hosting the node being queried.
	cassHostStateKey = kv.Key("db.cassandra.host.state")

	// cassBatchQueriesKey is the key for the attribute describing
	// the number of queries contained within the batch statement.
	cassBatchQueriesKey = kv.Key("db.cassandra.batch.queries")

	// cassErrMsgKey is the key for the attribute/label describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	cassErrMsgKey = kv.Key("db.cassandra.error.message")

	// cassRowsReturnedKey is the key for the span attribute describing the number of rows
	// returned on a query to the database.
	cassRowsReturnedKey = kv.Key("db.cassandra.rows.returned")

	// cassQueryAttemptsKey is the key for the span attribute describing the number of attempts
	// made for the query in question.
	cassQueryAttemptsKey = kv.Key("db.cassandra.attempts")

	// Static span names
	cassBatchQueryName = "Batch Query"
	cassConnectName    = "New Connection"

	// instrumentationName is the name of the instrumentation package.
	instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql"
)

// ------------------------------------------ Connection-level Attributes

// cassDBSystem returns the name of the DB system,
// cassandra, as a KeyValue pair (db.system).
func cassDBSystem() kv.KeyValue {
	return standard.DBSystemCassandra
}

// cassPeerName returns the hostname of the cassandra
// server as a standard KeyValue pair (net.peer.name).
func cassPeerName(name string) kv.KeyValue {
	return standard.NetPeerNameKey.String(name)
}

// cassPeerPort returns the port number of the cassandra
// server as a standard KeyValue pair (net.peer.port).
func cassPeerPort(port int) kv.KeyValue {
	return standard.NetPeerPortKey.Int(port)
}

// cassPeerIP returns the IP address of the cassandra
// server as a standard KeyValue pair (net.peer.ip).
func cassPeerIP(ip string) kv.KeyValue {
	return standard.NetPeerIPKey.String(ip)
}

// cassVersion returns the cql version as a KeyValue pair.
func cassVersion(version string) kv.KeyValue {
	return cassVersionKey.String(version)
}

// cassHostID returns the id of the cassandra host as a KeyValue pair.
func cassHostID(id string) kv.KeyValue {
	return cassHostIDKey.String(id)
}

// cassHostState returns the state of the cassandra host as a KeyValue pair.
func cassHostState(state string) kv.KeyValue {
	return cassHostStateKey.String(state)
}

// ------------------------------------------ Call-level attributes

// cassStatement returns the statement made to the cassandra database as a
// standard KeyValue pair (db.statement).
func cassStatement(stmt string) kv.KeyValue {
	return standard.DBStatementKey.String(stmt)
}

// cassDBOperation returns the batch query operation
// as a standard KeyValue pair (db.operation). This is used in lieu of a
// db.statement, which is not feasible to include in a span for a batch query
// because there can be n different query statements in a batch query.
func cassBatchQueryOperation() kv.KeyValue {
	cassBatchQueryOperation := "db.cassandra.batch.query"
	return standard.DBOperationKey.String(cassBatchQueryOperation)
}

// cassConnectOperation returns the connect operation
// as a standard KeyValue pair (db.operation). This is used in lieu of a
// db.statement since connection creation does not have a CQL statement.
func cassConnectOperation() kv.KeyValue {
	cassConnectOperation := "db.cassandra.connect"
	return standard.DBOperationKey.String(cassConnectOperation)
}

// cassKeyspace returns the keyspace of the session as
// a standard KeyValue pair (db.cassandra.keyspace).
func cassKeyspace(keyspace string) kv.KeyValue {
	return standard.DBCassandraKeyspaceKey.String(keyspace)
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
