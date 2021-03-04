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

package otelmongo

import "go.opentelemetry.io/otel/attribute"

const (
	DBApplicationKey = attribute.Key("db.application")
	DBNameKey        = attribute.Key("db.name")
	DBSystemKey      = attribute.Key("db.system")
	DBInstanceKey    = attribute.Key("db.instance")
	DBUserKey        = attribute.Key("db.user")
	DBStatementKey   = attribute.Key("db.statement")
)

// DBApplication indicates the application using the database.
func DBApplication(dbApplication string) attribute.KeyValue {
	return DBApplicationKey.String(dbApplication)
}

// DBName indicates the database name.
func DBName(dbName string) attribute.KeyValue {
	return DBNameKey.String(dbName)
}

// DBSystem indicates the system of Database.
func DBSystem(dbType string) attribute.KeyValue {
	return DBSystemKey.String(dbType)
}

// DBInstance indicates the instance name of Database.
func DBInstance(dbInstance string) attribute.KeyValue {
	return DBInstanceKey.String(dbInstance)
}

// DBUser indicates the user name of Database, e.g. "readonly_user" or "reporting_user".
func DBUser(dbUser string) attribute.KeyValue {
	return DBUserKey.String(dbUser)
}

// DBStatement records a database statement for the given database type.
func DBStatement(dbStatement string) attribute.KeyValue {
	return DBStatementKey.String(dbStatement)
}
