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

// Package mongo provides functions to trace the mongodb/mongo-go-driver package (https://github.com/mongodb/mongo-go-driver).
// It support v0.2.0 of github.com/mongodb/mongo-go-driver
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace requests.
package mongo

import "go.opentelemetry.io/otel/api/kv"

const (
	DBApplicationKey = kv.Key("db.application")
	DBNameKey        = kv.Key("db.name")
	DBTypeKey        = kv.Key("db.type")
	DBInstanceKey    = kv.Key("db.instance")
	DBUserKey        = kv.Key("db.user")
	DBStatementKey   = kv.Key("db.statement")
)

// DBApplication indicates the application using the database.
func DBApplication(dbApplication string) kv.KeyValue {
	return DBApplicationKey.String(dbApplication)
}

// DBName indicates the database name.
func DBName(dbName string) kv.KeyValue {
	return DBNameKey.String(dbName)
}

// DBType indicates the type of Database.
func DBType(dbType string) kv.KeyValue {
	return DBTypeKey.String(dbType)
}

// DBInstance indicates the instance name of Database.
func DBInstance(dbInstance string) kv.KeyValue {
	return DBInstanceKey.String(dbInstance)
}

// DBUser indicates the user name of Database, e.g. "readonly_user" or "reporting_user".
func DBUser(dbUser string) kv.KeyValue {
	return DBUserKey.String(dbUser)
}

// DBStatement records a database statement for the given database type.
func DBStatement(dbStatement string) kv.KeyValue {
	return DBStatementKey.String(dbStatement)
}
