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

package otelsql

import (
	"context"
	"time"

	otelcontrib "go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"
)

// The following tags are applied to stats recorded by this package.
var (
	// GoSQLInstance is the SQL instance name.
	GoSQLInstance = attribute.Key("go_sql_instance")
	// GoSQLMethod is the SQL method called.
	GoSQLMethod = attribute.Key("go_sql_method")
	// GoSQLError is the error received while calling a SQL method.
	GoSQLError = attribute.Key("go_sql_error")
	// GoSQLStatus identifies success vs. error from the SQL method response.
	GoSQLStatus = attribute.Key("go_sql_status")

	valueOK  = GoSQLStatus.String("OK")
	valueErr = GoSQLStatus.String("ERROR")
)

var (
	// InstrumentationName is the name of this instrumentation package.
	InstrumentationName = "go.sql"
	// InstrumentationVersion is the version of this instrumentation package.
	InstrumentationVersion = otelcontrib.SemVersion()
)

func Meter() metric.Meter {
	return global.Meter(InstrumentationName, metric.WithInstrumentationVersion(InstrumentationVersion))
}

// The following measures are supported for use in custom views.
var (
	MeasureLatencyMs = metric.Must(Meter()).NewInt64Histogram("go_sql_client_latency_milliseconds",
		metric.WithDescription("The latency of calls in milliseconds."),
		metric.WithUnit(unit.Milliseconds))
	MeasureOpenConnections = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_open",
		metric.WithDescription("Count of open connections in the pool."),
		metric.WithUnit(unit.Dimensionless))
	MeasureIdleConnections = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_idle",
		metric.WithDescription("Count of idle connections in the pool."),
		metric.WithUnit(unit.Dimensionless))
	MeasureActiveConnections = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_active",
		metric.WithDescription("Count of active connections in the pool."),
		metric.WithUnit(unit.Dimensionless))
	MeasureWaitCount = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_wait_count",
		metric.WithDescription("The total number of connections waited for."),
		metric.WithUnit(unit.Dimensionless))
	MeasureWaitDuration = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_wait_duration_milliseconds",
		metric.WithDescription("The total time blocked waiting for a new connection."),
		metric.WithUnit(unit.Milliseconds))
	MeasureIdleClosed = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_idle_closed",
		metric.WithDescription("The total number of connections closed due to SetMaxIdleConns."),
		metric.WithUnit(unit.Dimensionless))
	MeasureLifetimeClosed = metric.Must(Meter()).NewInt64Histogram("go_sql_connections_lifetime_closed",
		metric.WithDescription("The total number of connections closed due to SetConnMaxLifetime."),
		metric.WithUnit(unit.Dimensionless))
)

func recordCallStats(method, instanceName string) func(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	var labels = []attribute.KeyValue{
		GoSQLMethod.String(method),
		GoSQLInstance.String(instanceName),
	}
	startTime := time.Now()
	return func(ctx context.Context, err error, attrs ...attribute.KeyValue) {
		elapse := time.Since(startTime)
		if PostCall != nil {
			PostCall(ctx, err, elapse, attrs...)
		}
		timeSpentMs := elapse.Milliseconds()

		if err != nil {
			labels = append(labels, valueErr,
				GoSQLError.String(err.Error()))
		} else {
			labels = append(labels, valueOK)
		}

		MeasureLatencyMs.Record(ctx, timeSpentMs, labels...)
	}
}

// PostCall called after sql executed, designed such for logger to print details
var PostCall func(ctx context.Context, err error, elapse time.Duration, attrs ...attribute.KeyValue)
