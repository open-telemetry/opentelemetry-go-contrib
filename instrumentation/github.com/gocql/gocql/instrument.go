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

	// iQueryRows is the number of rows returned by a query.
	iQueryRows metric.Int64ValueRecorder

	// iBatchCount is the number of batch queries executed.
	iBatchCount metric.Int64Counter

	// iConnectionCount is the number of connections made
	// with the traced session.
	iConnectionCount metric.Int64Counter

	// iLatency is the sum of attempt latencies.
	iLatency metric.Int64ValueRecorder
)

// InstrumentWithProvider will recreate instruments using a meter
// from the given provider p.
func InstrumentWithProvider(p metric.Provider) {
	meter := p.Meter(instrumentationName)
	var err error

	if iQueryCount, err = meter.NewInt64Counter(
		"db.cassandra.queries",
		metric.WithDescription("Number queries executed"),
	); err != nil {
		log.Printf("failed to create iQueryCount instrument, %v", err)
	}

	if iQueryRows, err = meter.NewInt64ValueRecorder(
		"db.cassandra.rows",
		metric.WithDescription("Number of rows returned from query"),
	); err != nil {
		log.Printf("failed to create iQueryRows instrument, %v", err)
	}

	if iBatchCount, err = meter.NewInt64Counter(
		"db.cassandra.batch.queries",
		metric.WithDescription("Number of batch queries executed"),
	); err != nil {
		log.Printf("failed to create iBatchCount instrument, %v", err)
	}

	if iConnectionCount, err = meter.NewInt64Counter(
		"db.cassandra.connections",
		metric.WithDescription("Number of connections created"),
	); err != nil {
		log.Printf("failed to create iConnectionCount instrument, %v", err)
	}

	if iLatency, err = meter.NewInt64ValueRecorder(
		"db.cassandra.latency",
		metric.WithDescription("Sum of latency to host in milliseconds"),
		metric.WithUnit(unit.Milliseconds),
	); err != nil {
		log.Printf("failed to create iLatency instrument, %v", err)
	}
}

func init() {
	InstrumentWithProvider(global.MeterProvider())
}
