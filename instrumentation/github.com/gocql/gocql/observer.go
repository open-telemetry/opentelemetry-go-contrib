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

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"
)

// OtelQueryObserver implements the gocql.QueryObserver interface
// to provide instrumentation to gocql queries.
type OtelQueryObserver struct {
	observer gocql.QueryObserver
	cfg      *OtelConfig
}

// OtelBatchObserver implements the gocql.BatchObserver interface
// to provide instrumentation to gocql batch queries.
type OtelBatchObserver struct {
	observer gocql.BatchObserver
	cfg      *OtelConfig
}

// OtelConnectObserver implements the gocql.ConnectObserver interface
// to provide instrumentation to connection attempts made by the session.
type OtelConnectObserver struct {
	observer gocql.ConnectObserver
	cfg      *OtelConfig
	ctx      context.Context
}

// ----------------------------------------- Constructor Functions

// NewQueryObserver creates a QueryObserver that provides OpenTelemetry
// instrumentation for queries.
func NewQueryObserver(observer gocql.QueryObserver, cfg *OtelConfig) gocql.QueryObserver {
	return &OtelQueryObserver{
		observer,
		cfg,
	}
}

// NewBatchObserver creates a BatchObserver that provides OpenTelemetry instrumentation for
// batch queries.
func NewBatchObserver(observer gocql.BatchObserver, cfg *OtelConfig) gocql.BatchObserver {
	return &OtelBatchObserver{
		observer,
		cfg,
	}
}

// NewConnectObserver creates a ConnectObserver that provides OpenTelemetry instrumentation for
// connection attempts.
func NewConnectObserver(ctx context.Context, observer gocql.ConnectObserver, cfg *OtelConfig) gocql.ConnectObserver {
	return &OtelConnectObserver{
		observer,
		cfg,
		ctx,
	}
}

// ------------------------------------------ Observer Functions

// ObserveQuery is called once per query, and provides instrumentation for it.
func (o *OtelQueryObserver) ObserveQuery(ctx context.Context, observedQuery gocql.ObservedQuery) {
	if o.cfg.InstrumentQuery {
		host := observedQuery.Host
		keyspace := observedQuery.Keyspace

		attributes := includeKeyValues(host,
			cassKeyspace(keyspace),
			cassStatement(observedQuery.Statement),
			cassRowsReturned(observedQuery.Rows),
			cassQueryAttempts(observedQuery.Metrics.Attempts),
		)

		ctx, span := o.cfg.Tracer.Start(
			ctx,
			observedQuery.Statement,
			trace.WithStartTime(observedQuery.Start),
			trace.WithAttributes(attributes...),
		)

		if observedQuery.Err != nil {
			span.SetAttributes(cassErrMsg(observedQuery.Err.Error()))
			iQueryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassErrMsg(observedQuery.Err.Error()),
				)...,
			)
		} else {
			iQueryCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassStatement(observedQuery.Statement),
				)...,
			)
		}

		span.End(trace.WithEndTime(observedQuery.End))

		iQueryRows.Record(
			ctx,
			int64(observedQuery.Rows),
			includeKeyValues(host, cassKeyspace(keyspace))...,
		)
		iLatency.Record(
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
func (o *OtelBatchObserver) ObserveBatch(ctx context.Context, observedBatch gocql.ObservedBatch) {
	if o.cfg.InstrumentBatch {
		host := observedBatch.Host
		keyspace := observedBatch.Keyspace

		attributes := includeKeyValues(host,
			cassKeyspace(keyspace),
			cassBatchQueryOperation(),
			cassBatchQueries(len(observedBatch.Statements)),
		)

		ctx, span := o.cfg.Tracer.Start(
			ctx,
			cassBatchQueryName,
			trace.WithStartTime(observedBatch.Start),
			trace.WithAttributes(attributes...),
		)

		if observedBatch.Err != nil {
			span.SetAttributes(cassErrMsg(observedBatch.Err.Error()))
			iBatchCount.Add(
				ctx,
				1,
				includeKeyValues(host,
					cassKeyspace(keyspace),
					cassErrMsg(observedBatch.Err.Error()),
				)...,
			)
		} else {
			iBatchCount.Add(
				ctx,
				1,
				includeKeyValues(host, cassKeyspace(keyspace))...,
			)
		}

		span.End(trace.WithEndTime(observedBatch.End))

		iLatency.Record(
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
func (o *OtelConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	if o.cfg.InstrumentConnect {
		host := observedConnect.Host

		attributes := includeKeyValues(host, cassConnectOperation())

		_, span := o.cfg.Tracer.Start(
			o.ctx,
			cassConnectName,
			trace.WithStartTime(observedConnect.Start),
			trace.WithAttributes(attributes...),
		)

		if observedConnect.Err != nil {
			span.SetAttributes(cassErrMsg(observedConnect.Err.Error()))
			iConnectionCount.Add(
				o.ctx,
				1,
				includeKeyValues(host, cassErrMsg(observedConnect.Err.Error()))...,
			)
		} else {
			iConnectionCount.Add(
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
func includeKeyValues(host *gocql.HostInfo, values ...kv.KeyValue) []kv.KeyValue {
	connectionLevelAttributes := []kv.KeyValue{
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
func hostOrIP(hostnameAndPort string) kv.KeyValue {
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
