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

	"github.com/gocql/gocql"
)

// NewSessionWithTracing creates a new session using the given cluster
// configuration enabling tracing for queries, batch queries, and connection attempts.
// You may use additional observers and disable specific tracing using the provided `TracedSessionOption`s.
func NewSessionWithTracing(ctx context.Context, cluster *gocql.ClusterConfig, options ...TracedSessionOption) (*gocql.Session, error) {
	config := configure(options...)
	instruments := newInstruments(config.meterProvider)
	tracer := config.traceProvider.Tracer(instrumentationName)
	cluster.QueryObserver = &OTelQueryObserver{
		enabled:  config.instrumentQuery,
		observer: config.queryObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	cluster.BatchObserver = &OTelBatchObserver{
		enabled:  config.instrumentBatch,
		observer: config.batchObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	cluster.ConnectObserver = &OTelConnectObserver{
		ctx:      ctx,
		enabled:  config.instrumentConnect,
		observer: config.connectObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	return cluster.CreateSession()
}
