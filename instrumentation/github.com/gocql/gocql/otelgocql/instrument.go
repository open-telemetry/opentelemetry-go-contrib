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

package otelgocql // import "go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql"

import (
	"log"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
)

type instruments struct {
	// queryCount is the number of queries executed.
	queryCount instrument.Int64Counter

	// queryRows is the number of rows returned by a query.
	queryRows instrument.Int64Histogram

	// batchCount is the number of batch queries executed.
	batchCount instrument.Int64Counter

	// connectionCount is the number of connections made
	// with the traced session.
	connectionCount instrument.Int64Counter

	// latency is the sum of attempt latencies.
	latency instrument.Int64Histogram
}

// newInstruments will create instruments using a meter
// from the given provider p.
func newInstruments(p metric.MeterProvider) *instruments {
	meter := p.Meter(
		internal.InstrumentationName,
		metric.WithInstrumentationVersion(SemVersion()),
	)
	instruments := &instruments{}
	var err error

	if instruments.queryCount, err = meter.Int64Counter(
		"db.cassandra.queries",
		instrument.WithDescription("Number queries executed"),
	); err != nil {
		log.Printf("failed to create iQueryCount instrument, %v", err)
	}

	if instruments.queryRows, err = meter.Int64Histogram(
		"db.cassandra.rows",
		instrument.WithDescription("Number of rows returned from query"),
	); err != nil {
		log.Printf("failed to create iQueryRows instrument, %v", err)
	}

	if instruments.batchCount, err = meter.Int64Counter(
		"db.cassandra.batch.queries",
		instrument.WithDescription("Number of batch queries executed"),
	); err != nil {
		log.Printf("failed to create iBatchCount instrument, %v", err)
	}

	if instruments.connectionCount, err = meter.Int64Counter(
		"db.cassandra.connections",
		instrument.WithDescription("Number of connections created"),
	); err != nil {
		log.Printf("failed to create iConnectionCount instrument, %v", err)
	}

	if instruments.latency, err = meter.Int64Histogram(
		"db.cassandra.latency",
		instrument.WithDescription("Sum of latency to host in milliseconds"),
		instrument.WithUnit("ms"),
	); err != nil {
		log.Printf("failed to create iLatency instrument, %v", err)
	}

	return instruments
}
