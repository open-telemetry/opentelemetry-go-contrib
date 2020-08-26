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
	"net"
	"time"

	"github.com/gocql/gocql"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
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
			cassKeyspace(keyspace),
			cassStatement(observedQuery.Statement),
			cassRowsReturned(observedQuery.Rows),
			cassQueryAttempts(observedQuery.Metrics.Attempts),
		)

		ctx, span := o.tracer.Start(
			ctx,
			observedQuery.Statement,
			trace.WithStartTime(observedQuery.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedQuery.Err != nil {
			span.SetAttributes(cassErrMsg(observedQuery.Err.Error()))
			inst.queryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassStatement(observedQuery.Statement),
					cassErrMsg(observedQuery.Err.Error()),
				)...,
			)
		} else {
			inst.queryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassStatement(observedQuery.Statement),
				)...,
			)
		}

		span.End(trace.WithEndTime(observedQuery.End))

		inst.queryRows.Record(
			ctx,
			int64(observedQuery.Rows),
			includeKeyValues(host, cassKeyspace(keyspace))...,
		)
		inst.latency.Record(
			ctx,
			nanoToMilliseconds(observedQuery.Metrics.TotalLatency),
			includeKeyValues(host, cassKeyspace(keyspace))...,
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
			cassKeyspace(keyspace),
			cassBatchQueryOperation(),
			cassBatchQueries(len(observedBatch.Statements)),
		)

		ctx, span := o.tracer.Start(
			ctx,
			cassBatchQueryName,
			trace.WithStartTime(observedBatch.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedBatch.Err != nil {
			span.SetAttributes(cassErrMsg(observedBatch.Err.Error()))
			inst.batchCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassErrMsg(observedBatch.Err.Error()),
				)...,
			)
		} else {
			inst.batchCount.Add(
				ctx,
				1,
				includeKeyValues(host, cassKeyspace(keyspace))...,
			)
		}

		span.End(trace.WithEndTime(observedBatch.End))

		inst.latency.Record(
			ctx,
			nanoToMilliseconds(observedBatch.Metrics.TotalLatency),
			includeKeyValues(host, cassKeyspace(keyspace))...,
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

		attributes := includeKeyValues(host, cassConnectOperation())

		_, span := o.tracer.Start(
			o.ctx,
			cassConnectName,
			trace.WithStartTime(observedConnect.Start),
			trace.WithAttributes(attributes...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		if observedConnect.Err != nil {
			span.SetAttributes(cassErrMsg(observedConnect.Err.Error()))
			inst.connectionCount.Add(
				o.ctx,
				1,
				includeKeyValues(host, cassErrMsg(observedConnect.Err.Error()))...,
			)
		} else {
			inst.connectionCount.Add(
				o.ctx,
				1,
				includeKeyValues(host)...,
			)
		}

		span.End(trace.WithEndTime(observedConnect.End))
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
func includeKeyValues(host *gocql.HostInfo, values ...label.KeyValue) []label.KeyValue {
	connectionLevelAttributes := []label.KeyValue{
		cassDBSystem(),
		hostOrIP(host.HostnameAndPort()),
		cassPeerPort(host.Port()),
		cassVersion(host.Version().String()),
		cassHostID(host.HostID()),
		cassHostState(host.State().String()),
	}
	return append(connectionLevelAttributes, values...)
}

// hostOrIP returns a KeyValue pair for the hostname
// retrieved from gocql.HostInfo.HostnameAndPort(). If the hostname
// is returned as a resolved IP address (as is the case for localhost),
// then the KeyValue will have the key net.peer.ip.
// If the hostname is the proper DNS name, then the key will be net.peer.name.
func hostOrIP(hostnameAndPort string) label.KeyValue {
	hostname, _, err := net.SplitHostPort(hostnameAndPort)
	if err != nil {
		log.Printf("failed to parse hostname from port, %v", err)
	}
	if parse := net.ParseIP(hostname); parse != nil {
		return cassPeerIP(parse.String())
	}
	return cassPeerName(hostname)
}

// nanoToMilliseconds converts nanoseconds to milliseconds.
func nanoToMilliseconds(ns int64) int64 {
	return ns / int64(time.Millisecond)
}
