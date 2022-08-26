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

package internal // import "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"

import (
	"log"
	"net"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const (
	// CassVersionKey is the key for the attribute/label describing
	// the cql version.
	CassVersionKey = attribute.Key("db.cassandra.version")

	// CassHostIDKey is the key for the attribute/label describing the id
	// of the host being queried.
	CassHostIDKey = attribute.Key("db.cassandra.host.id")

	// CassHostStateKey is the key for the attribute/label describing
	// the state of the casssandra server hosting the node being queried.
	CassHostStateKey = attribute.Key("db.cassandra.host.state")

	// CassBatchQueriesKey is the key for the attribute describing
	// the number of queries contained within the batch statement.
	CassBatchQueriesKey = attribute.Key("db.cassandra.batch.queries")

	// CassErrMsgKey is the key for the attribute/label describing
	// the error message from an error encountered when executing a query, batch,
	// or connection attempt to the cassandra server.
	CassErrMsgKey = attribute.Key("db.cassandra.error.message")

	// CassRowsReturnedKey is the key for the span attribute describing the number of rows
	// returned on a query to the database.
	CassRowsReturnedKey = attribute.Key("db.cassandra.rows.returned")

	// CassQueryAttemptsKey is the key for the span attribute describing the number of attempts
	// made for the query in question.
	CassQueryAttemptsKey = attribute.Key("db.cassandra.attempts")

	// CassBatchQueryName is the batch operation span name.
	CassBatchQueryName = "Batch Query"
	// CassConnectName is the connect operation span name.
	CassConnectName = "New Connection"

	// InstrumentationName is the name of the instrumentation package.
	InstrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql"
)

// ------------------------------------------ Connection-level Attributes

// CassDBSystem returns the name of the DB system,
// cassandra, as a KeyValue pair (db.system).
func CassDBSystem() attribute.KeyValue {
	return semconv.DBSystemCassandra
}

// CassPeerName returns the hostname of the cassandra
// server as a semconv KeyValue pair (net.peer.name).
func CassPeerName(name string) attribute.KeyValue {
	return semconv.NetPeerNameKey.String(name)
}

// CassPeerPort returns the port number of the cassandra
// server as a semconv KeyValue pair (net.peer.port).
func CassPeerPort(port int) attribute.KeyValue {
	return semconv.NetPeerPortKey.Int(port)
}

// CassPeerIP returns the IP address of the cassandra
// server as a semconv KeyValue pair (net.peer.ip).
func CassPeerIP(ip string) attribute.KeyValue {
	return semconv.NetPeerIPKey.String(ip)
}

// CassVersion returns the cql version as a KeyValue pair.
func CassVersion(version string) attribute.KeyValue {
	return CassVersionKey.String(version)
}

// CassHostID returns the id of the cassandra host as a KeyValue pair.
func CassHostID(id string) attribute.KeyValue {
	return CassHostIDKey.String(id)
}

// CassHostState returns the state of the cassandra host as a KeyValue pair.
func CassHostState(state string) attribute.KeyValue {
	return CassHostStateKey.String(state)
}

// ------------------------------------------ Call-level attributes

// CassStatement returns the statement made to the cassandra database as a
// semconv KeyValue pair (db.statement).
func CassStatement(stmt string) attribute.KeyValue {
	return semconv.DBStatementKey.String(stmt)
}

// CassBatchQueryOperation returns the batch query operation
// as a semconv KeyValue pair (db.operation). This is used in lieu of a
// db.statement, which is not feasible to include in a span for a batch query
// because there can be n different query statements in a batch query.
func CassBatchQueryOperation() attribute.KeyValue {
	cassBatchQueryOperation := "db.cassandra.batch.query"
	return semconv.DBOperationKey.String(cassBatchQueryOperation)
}

// CassConnectOperation returns the connect operation
// as a semconv KeyValue pair (db.operation). This is used in lieu of a
// db.statement since connection creation does not have a CQL statement.
func CassConnectOperation() attribute.KeyValue {
	cassConnectOperation := "db.cassandra.connect"
	return semconv.DBOperationKey.String(cassConnectOperation)
}

// CassKeyspace returns the keyspace of the session as
// a semconv KeyValue pair (db.name).
func CassKeyspace(keyspace string) attribute.KeyValue {
	return semconv.DBNameKey.String(keyspace)
}

// CassBatchQueries returns the number of queries in a batch query
// as a KeyValue pair.
func CassBatchQueries(num int) attribute.KeyValue {
	return CassBatchQueriesKey.Int(num)
}

// CassErrMsg returns the KeyValue pair of an error message
// encountered when executing a query, batch query, or error.
func CassErrMsg(msg string) attribute.KeyValue {
	return CassErrMsgKey.String(msg)
}

// CassRowsReturned returns the KeyValue pair of the number of rows
// returned from a query.
func CassRowsReturned(rows int) attribute.KeyValue {
	return CassRowsReturnedKey.Int(rows)
}

// CassQueryAttempts returns the KeyValue pair of the number of attempts
// made for a query.
func CassQueryAttempts(num int) attribute.KeyValue {
	return CassQueryAttemptsKey.Int(num)
}

// HostOrIP returns a KeyValue pair for the hostname
// retrieved from gocql.HostInfo.HostnameAndPort(). If the hostname
// is returned as a resolved IP address (as is the case for localhost),
// then the KeyValue will have the key net.peer.ip.
// If the hostname is the proper DNS name, then the key will be net.peer.name.
func HostOrIP(hostnameAndPort string) attribute.KeyValue {
	hostname, _, err := net.SplitHostPort(hostnameAndPort)
	if err != nil {
		log.Printf("failed to parse hostname from port, %v", err)
	}
	if parse := net.ParseIP(hostname); parse != nil {
		return CassPeerIP(parse.String())
	}
	return CassPeerName(hostname)
}
