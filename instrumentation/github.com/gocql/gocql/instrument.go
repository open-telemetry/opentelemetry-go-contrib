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
	"log"

	"go.opentelemetry.io/otel/api/global"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
)

var (
	// iQueryCount is the number of queries executed.
	iQueryCount metric.Int64Counter

	// iQueryErrors is the number of errors encountered when
	// executing queries.
	iQueryErrors metric.Int64Counter

	// iQueryRows is the number of rows returned by a query.
	iQueryRows metric.Int64ValueRecorder

	// iBatchCount is the number of batch queries executed.
	iBatchCount metric.Int64Counter

	// iBatchErrors is the number of errors encountered when
	// executing batch queries.
	iBatchErrors metric.Int64Counter

	// iConnectionCount is the number of connections made
	// with the traced session.
	iConnectionCount metric.Int64Counter

	// iConnectErrors is the number of errors encountered
	// when making connections with the current traced session.
	iConnectErrors metric.Int64Counter

	// iLatency is the sum of attempt latencies.
	iLatency metric.Int64ValueRecorder
)

// InstrumentWithProvider will recreate instruments using a meter
// from the given provider p.
func InstrumentWithProvider(p metric.Provider) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Print("failed to create meter. metrics are not being recorded")
		}
	}()
	meter := metric.Must(p.Meter("github.com/gocql/gocql"))

	iQueryCount = meter.NewInt64Counter(
		"cassandra.queries",
		metric.WithDescription("Number queries executed"),
	)

	iQueryErrors = meter.NewInt64Counter(
		"cassandra.query.errors",
		metric.WithDescription("Number of errors encountered when executing queries"),
	)

	iQueryRows = meter.NewInt64ValueRecorder(
		"cassandra.rows",
		metric.WithDescription("Number of rows returned from query"),
	)

	iBatchCount = meter.NewInt64Counter(
		"cassandra.batch.queries",
		metric.WithDescription("Number of batch queries executed"),
	)

	iBatchErrors = meter.NewInt64Counter(
		"cassandra.batch.errors",
		metric.WithDescription("Number of errors encountered when executing batch queries"),
	)

	iConnectionCount = meter.NewInt64Counter(
		"cassandra.connections",
		metric.WithDescription("Number of connections created"),
	)

	iConnectErrors = meter.NewInt64Counter(
		"cassandra.connect.errors",
		metric.WithDescription("Number of errors encountered when creating connections"),
	)

	iLatency = meter.NewInt64ValueRecorder(
		"cassandra.latency",
		metric.WithDescription("Sum of latency to host in milliseconds"),
		metric.WithUnit(unit.Milliseconds),
	)
}

func init() {
	InstrumentWithProvider(global.MeterProvider())
}
