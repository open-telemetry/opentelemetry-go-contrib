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
		attributes := append(
			defaultAttributes(observedQuery.Host),
			CassStatement(observedQuery.Statement),
		)

		if observedQuery.Err != nil {
			attributes = append(
				attributes,
				CassErrMsg(observedQuery.Err.Error()),
			)
			iQueryErrors.Add(ctx, 1)
		}

		ctx, span := o.cfg.tracer.Start(
			ctx,
			cassQueryName,
			trace.WithStartTime(observedQuery.Start),
			trace.WithAttributes(attributes...),
		)

		span.End(trace.WithEndTime(observedQuery.End))

		iQueryCount.Add(ctx, 1, CassStatement(observedQuery.Statement))
		iQueryRows.Record(ctx, int64(observedQuery.Rows))
	}
	if o.observer != nil {
		o.observer.ObserveQuery(ctx, observedQuery)
	}
}

// ObserveBatch instruments a specific batch query.
func (o *OtelBatchObserver) ObserveBatch(ctx context.Context, observedBatch gocql.ObservedBatch) {
	if o.cfg.instrumentBatch {
		attributes := append(
			defaultAttributes(observedBatch.Host),
			CassBatchStatements(observedBatch.Statements),
		)

		if observedBatch.Err != nil {
			attributes = append(
				attributes,
				CassErrMsg(observedBatch.Err.Error()),
			)
			iBatchErrors.Add(ctx, 1)
		}

		ctx, span := o.cfg.tracer.Start(
			ctx,
			cassBatchQueryName,
			trace.WithStartTime(observedBatch.Start),
			trace.WithAttributes(attributes...),
		)

		span.End(trace.WithEndTime(observedBatch.End))

		iBatchCount.Add(ctx, 1)
	}

	if o.observer != nil {
		o.observer.ObserveBatch(ctx, observedBatch)
	}
}

// ObserveConnect instruments a specific connection attempt.
func (o *OtelConnectObserver) ObserveConnect(observedConnect gocql.ObservedConnect) {
	if o.cfg.instrumentConnect {
		attributes := defaultAttributes(observedConnect.Host)

		if observedConnect.Err != nil {
			attributes = append(
				attributes,
				CassErrMsg(observedConnect.Err.Error()),
			)
			iConnectErrors.Add(o.ctx, 1)
		}

		_, span := o.cfg.tracer.Start(
			o.ctx,
			cassConnectName,
			trace.WithStartTime(observedConnect.Start),
			trace.WithAttributes(attributes...),
		)

		span.End(trace.WithEndTime(observedConnect.End))

		host := observedConnect.Host.HostnameAndPort()
		iConnectionCount.Add(o.ctx, 1, CassHostKey.String(host))
	}

	if o.observer != nil {
		o.observer.ObserveConnect(observedConnect)
	}
}

// ------------------------------------------ Private Functions

func defaultAttributes(host *gocql.HostInfo) []kv.KeyValue {
	hostnameAndPort := host.HostnameAndPort()
	return []kv.KeyValue{
		CassVersion(host.Version().String()),
		CassHost(hostnameAndPort[0:strings.Index(hostnameAndPort, ":")]),
		CassPort(host.Port()),
		CassHostState(host.State().String()),
	}
}
