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
	"time"

	"github.com/gocql/gocql"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OTelQueryObserver implements the gocql.QueryObserver interface
// to provide instrumentation to gocql queries.
type OTelQueryObserver struct {
	enabled  bool
	observer gocql.QueryObserver
	tracer   trace.Tracer
	inst     *instruments
}

// OTelBatchObserver implements the gocql.BatchObserver interface
// to provide instrumentation to gocql batch queries.
type OTelBatchObserver struct {
	enabled  bool
	observer gocql.BatchObserver
	tracer   trace.Tracer
	inst     *instruments
}

// OTelConnectObserver implements the gocql.ConnectObserver interface
// to provide instrumentation to connection attempts made by the session.
type OTelConnectObserver struct {
	ctx      context.Context
	enabled  bool
	observer gocql.ConnectObserver
	tracer   trace.Tracer
	inst     *instruments
}

// ------------------------------------------ Observer Functions

// ObserveQuery is called once per query, and provides instrumentation for it.
func (o *OTelQueryObserver) ObserveQuery(ctx context.Context, observedQuery gocql.ObservedQuery) {
	if o.enabled {
		host := observedQuery.Host
		keyspace := observedQuery.Keyspace
		inst := o.inst

		attributes := includeKeyValues(host,
			internal.CassKeyspace(keyspace),
			internal.CassStatement(observedQuery.Statement),
			internal.CassRowsReturned(observedQuery.Rows),
			internal.CassQueryAttempts(observedQuery.Metrics.Attempts),
		)

		ctx, span := o.tracer.Start(
			ctx,
			observedQuery.Statement,
			trace.WithTimestamp(observedQuery.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedQuery.Err != nil {
			span.SetAttributes(internal.CassErrMsg(observedQuery.Err.Error()))
			inst.queryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					internal.CassKeyspace(keyspace),
					internal.CassStatement(observedQuery.Statement),
					internal.CassErrMsg(observedQuery.Err.Error()),
				)...,
			)
		} else {
			inst.queryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					internal.CassKeyspace(keyspace),
					internal.CassStatement(observedQuery.Statement),
				)...,
			)
		}

		span.End(trace.WithTimestamp(observedQuery.End))

		inst.queryRows.Record(
			ctx,
			int64(observedQuery.Rows),
			includeKeyValues(host, internal.CassKeyspace(keyspace))...,
		)
		inst.latency.Record(
			ctx,
			nanoToMilliseconds(observedQuery.Metrics.TotalLatency),
			includeKeyValues(host, internal.CassKeyspace(keyspace))...,
		)
	}

	if o.observer != nil {
		o.observer.ObserveQuery(ctx, observedQuery)
	}
}

// ObserveBatch is called once per batch query, and provides instrumentation for it.
func (o *OTelBatchObserver) ObserveBatch(ctx context.Context, observedBatch gocql.ObservedBatch) {
	if o.enabled {
		host := observedBatch.Host
		keyspace := observedBatch.Keyspace
		inst := o.inst

		attributes := includeKeyValues(host,
			internal.CassKeyspace(keyspace),
			internal.CassBatchQueryOperation(),
			internal.CassBatchQueries(len(observedBatch.Statements)),
		)

		ctx, span := o.tracer.Start(
			ctx,
			internal.CassBatchQueryName,
			trace.WithTimestamp(observedBatch.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedBatch.Err != nil {
			span.SetAttributes(internal.CassErrMsg(observedBatch.Err.Error()))
			inst.batchCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					internal.CassKeyspace(keyspace),
					internal.CassErrMsg(observedBatch.Err.Error()),
				)...,
			)
		} else {
			inst.batchCount.Add(
				ctx,
				1,
				includeKeyValues(host, internal.CassKeyspace(keyspace))...,
			)
		}

		span.End(trace.WithTimestamp(observedBatch.End))

		inst.latency.Record(
			ctx,
			nanoToMilliseconds(observedBatch.Metrics.TotalLatency),
			includeKeyValues(host, internal.CassKeyspace(keyspace))...,
		)
	}

	if o.observer != nil {
		o.observer.ObserveBatch(ctx, observedBatch)
	}
}

// ObserveConnect is called once per connection attempt, and provides instrumentation for it.
func (o *OTelConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	if o.enabled {
		host := observedConnect.Host
		inst := o.inst

		attributes := includeKeyValues(host, internal.CassConnectOperation())

		_, span := o.tracer.Start(
			o.ctx,
			internal.CassConnectName,
			trace.WithTimestamp(observedConnect.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedConnect.Err != nil {
			span.SetAttributes(internal.CassErrMsg(observedConnect.Err.Error()))
			inst.connectionCount.Add(
				o.ctx,
				1,
				includeKeyValues(host, internal.CassErrMsg(observedConnect.Err.Error()))...,
			)
		} else {
			inst.connectionCount.Add(
				o.ctx,
				1,
				includeKeyValues(host)...,
			)
		}

		span.End(trace.WithTimestamp(observedConnect.End))
	}

	if o.observer != nil {
		o.observer.ObserveConnect(observedConnect)
	}
}

// ------------------------------------------ Private Functions

// includeKeyValues is a convenience function for adding multiple attributes/labels to a
// span or instrument. By default, this function includes connection-level attributes,
// (as per the semantic conventions) which have been made standard for all spans and metrics
// generated by this instrumentation integration.
func includeKeyValues(host *gocql.HostInfo, values ...attribute.KeyValue) []attribute.KeyValue {
	connectionLevelAttributes := []attribute.KeyValue{
		internal.CassDBSystem(),
		internal.HostOrIP(host.HostnameAndPort()),
		internal.CassPeerPort(host.Port()),
		internal.CassVersion(host.Version().String()),
		internal.CassHostID(host.HostID()),
		internal.CassHostState(host.State().String()),
	}
	return append(connectionLevelAttributes, values...)
}

// nanoToMilliseconds converts nanoseconds to milliseconds.
func nanoToMilliseconds(ns int64) int64 {
	return ns / int64(time.Millisecond)
}
