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
	"context"

	"github.com/gocql/gocql"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"
	"go.opentelemetry.io/otel/trace"
)

// NewSessionWithTracing creates a new session using the given cluster
// configuration enabling tracing for queries, batch queries, and connection attempts.
// You may use additional observers and disable specific tracing using the provided `TracedSessionOption`s.
func NewSessionWithTracing(ctx context.Context, cluster *gocql.ClusterConfig, options ...Option) (*gocql.Session, error) {
	cfg := newConfig(options...)
	instruments := newInstruments(cfg.meterProvider)
	tracer := cfg.tracerProvider.Tracer(
		internal.InstrumentationName,
		trace.WithInstrumentationVersion(SemVersion()),
	)
	cluster.QueryObserver = &OTelQueryObserver{
		enabled:  cfg.instrumentQuery,
		observer: cfg.queryObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	cluster.BatchObserver = &OTelBatchObserver{
		enabled:  cfg.instrumentBatch,
		observer: cfg.batchObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	cluster.ConnectObserver = &OTelConnectObserver{
		ctx:      ctx,
		enabled:  cfg.instrumentConnect,
		observer: cfg.connectObserver,
		tracer:   tracer,
		inst:     instruments,
	}
	return cluster.CreateSession()
}
