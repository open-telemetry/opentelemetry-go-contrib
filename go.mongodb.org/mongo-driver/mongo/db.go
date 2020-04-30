// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.
// Copyright 2020 The OpenTelemetry Authors

// Package mongo provides functions to trace the mongodb/mongo-go-driver package (https://github.com/mongodb/mongo-go-driver).
// It support v0.2.0 of github.com/mongodb/mongo-go-driver
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace requests.
package mongo

import "go.opentelemetry.io/otel/api/core"

const (
	DBApplicationKey = core.Key("db.application")
	DBNameKey        = core.Key("db.name")
	DBTypeKey        = core.Key("db.type")
	DBInstanceKey    = core.Key("db.instance")
	DBUserKey        = core.Key("db.user")
	DBStatementKey   = core.Key("db.statement")
)

// DBApplication indicates the application using the database.
func DBApplication(dbApplication string) core.KeyValue {
	return DBApplicationKey.String(dbApplication)
}

// DBName indicates the database name.
func DBName(dbName string) core.KeyValue {
	return DBNameKey.String(dbName)
}

// DBType indicates the type of Database.
func DBType(dbType string) core.KeyValue {
	return DBTypeKey.String(dbType)
}

// DBInstance indicates the instance name of Database.
func DBInstance(dbInstance string) core.KeyValue {
	return DBInstanceKey.String(dbInstance)
}

// DBUser indicates the user name of Database, e.g. "readonly_user" or "reporting_user".
func DBUser(dbUser string) core.KeyValue {
	return DBUserKey.String(dbUser)
}

// DBStatement records a database statement for the given database type.
func DBStatement(dbStatement string) core.KeyValue {
	return DBStatementKey.String(dbStatement)
}
