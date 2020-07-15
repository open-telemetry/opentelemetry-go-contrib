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
	"strings"
	"time"

	"github.com/gocql/gocql"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
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
// tracing and metrics.
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

// ObserveQuery instruments a specific query.
func (o *OtelQueryObserver) ObserveQuery(ctx context.Context, observedQuery gocql.ObservedQuery) {
	if o.cfg.instrumentQuery {
		host := observedQuery.Host
		keyspace := observedQuery.Keyspace

		attributes := append(
			defaultAttributes(host),
			CassStatement(observedQuery.Statement),
			CassRowsReturned(observedQuery.Rows),
			CassQueryAttempts(observedQuery.Metrics.Attempts),
			CassQueryAttemptNum(observedQuery.Attempt),
			CassKeyspace(keyspace),
		)

		ctx, span := o.cfg.tracer.Start(
			ctx,
			cassQueryName,
			trace.WithStartTime(observedQuery.Start),
			trace.WithAttributes(attributes...),
		)

		if observedQuery.Err != nil {
			span.SetAttributes(CassErrMsg(observedQuery.Err.Error()))
			recordError(ctx, iQueryErrors, keyspace, host)
		}

		span.End(trace.WithEndTime(observedQuery.End))

		queryLabels := append(
			defaultMetricLabels(keyspace, host),
			CassStatement(observedQuery.Statement),
		)
		iQueryCount.Add(
			ctx,
			1,
			queryLabels...,
		)
		iQueryRows.Record(
			ctx,
			int64(observedQuery.Rows),
			defaultMetricLabels(keyspace, host)...,
		)
		iLatency.Record(
			ctx,
			nanoToMilliseconds(observedQuery.Metrics.TotalLatency),
			defaultMetricLabels(keyspace, host)...,
		)
	}

	if o.observer != nil {
		o.observer.ObserveQuery(ctx, observedQuery)
	}
}

// ObserveBatch instruments a specific batch query.
func (o *OtelBatchObserver) ObserveBatch(ctx context.Context, observedBatch gocql.ObservedBatch) {
	if o.cfg.instrumentBatch {
		host := observedBatch.Host
		keyspace := observedBatch.Keyspace
		attributes := append(
			defaultAttributes(host),
			CassBatchQueries(len(observedBatch.Statements)),
			CassKeyspace(keyspace),
		)

		ctx, span := o.cfg.tracer.Start(
			ctx,
			cassBatchQueryName,
			trace.WithStartTime(observedBatch.Start),
			trace.WithAttributes(attributes...),
		)

		if observedBatch.Err != nil {
			span.SetAttributes(CassErrMsg(observedBatch.Err.Error()))
			recordError(ctx, iBatchErrors, keyspace, host)
		}

		span.End(trace.WithEndTime(observedBatch.End))

		iBatchCount.Add(
			ctx,
			1,
			defaultMetricLabels(observedBatch.Keyspace, observedBatch.Host)...,
		)
		iLatency.Record(
			ctx,
			nanoToMilliseconds(observedBatch.Metrics.TotalLatency),
			defaultMetricLabels(keyspace, host)...,
		)
	}

	if o.observer != nil {
		o.observer.ObserveBatch(ctx, observedBatch)
	}
}

// ObserveConnect instruments a specific connection attempt.
func (o *OtelConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	if o.cfg.instrumentConnect {
		host := observedConnect.Host
		hostname := getHost(host.HostnameAndPort())
		attributes := defaultAttributes(observedConnect.Host)

		_, span := o.cfg.tracer.Start(
			o.ctx,
			cassConnectName,
			trace.WithStartTime(observedConnect.Start),
			trace.WithAttributes(attributes...),
		)

		if observedConnect.Err != nil {
			span.SetAttributes(CassErrMsg(observedConnect.Err.Error()))
			iConnectErrors.Add(
				o.ctx,
				1,
				CassHost(hostname),
				CassHostID(host.HostID()),
			)
		}

		span.End(trace.WithEndTime(observedConnect.End))

		iConnectionCount.Add(
			o.ctx,
			1,
			CassHost(hostname),
			CassHostID(host.HostID()),
		)
	}

	if o.observer != nil {
		o.observer.ObserveConnect(observedConnect)
	}
}

// ------------------------------------------ Private Functions

// getHost returns the hostname as a string.
// gocql.HostInfo.HostnameAndPort() returns a string
// formatted like host:port. This function returns the host.
func getHost(hostPort string) string {
	idx := strings.Index(hostPort, ":")
	host := hostPort[0:idx]
	return host
}

// defaultAttributes creates an array of KeyValue pairs that are
// attributes for all gocql spans.
func defaultAttributes(host *gocql.HostInfo) []kv.KeyValue {
	hostnameAndPort := host.HostnameAndPort()
	return []kv.KeyValue{
		CassVersion(host.Version().String()),
		CassHost(getHost(hostnameAndPort)),
		CassPort(host.Port()),
		CassHostState(host.State().String()),
		CassHostID(host.HostID()),
	}
}

// defaultMetricLabels returns an array of the default labels added to metrics.
func defaultMetricLabels(keyspace string, host *gocql.HostInfo) []kv.KeyValue {
	return []kv.KeyValue{
		CassHostID(host.HostID()),
		CassKeyspace(keyspace),
	}
}

// nanoToMilliseconds converts nanoseconds to milliseconds.
func nanoToMilliseconds(ns int64) int64 {
	return ns / int64(time.Millisecond)
}

func recordError(ctx context.Context, counter metric.Int64Counter, keyspace string, host *gocql.HostInfo) {
	labels := append(defaultMetricLabels(keyspace, host), CassHostState(host.State().String()))
	counter.Add(
		ctx,
		1,
		labels...,
	)
}
