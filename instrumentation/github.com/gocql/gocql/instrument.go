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
	"context"
	"log"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
)

var (
	// Query
	iQueryCount  metric.Int64Counter
	iQueryErrors metric.Int64Counter

	// Batch
	iBatchCount  metric.Int64Counter
	iBatchErrors metric.Int64Counter

	// Connections
	iConnectionCount metric.Int64Counter
	iConnectErrors   metric.Int64Counter

	iLatency metric.Int64ValueRecorder
)

func init() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Print("failed to create meter. metrics are not being recorded")
		}
	}()
	meter := metric.Must(global.Meter("github.com/gocql/gocql"))

	iQueryCount = meter.NewInt64Counter("cassandra.queries")
	iQueryErrors = meter.NewInt64Counter("cassandra.query_errors")

	iBatchCount = meter.NewInt64Counter("cassandra.batch_queries")
	iBatchErrors = meter.NewInt64Counter("cassandra.batch_errors")

	iConnectionCount = meter.NewInt64Counter("cassandra.connections")
	iConnectErrors = meter.NewInt64Counter("cassandra.connect_errors")
}

func countQuery(ctx context.Context, stmt string) {
	iQueryCount.Add(ctx, 1, CassStatement(stmt))
}
