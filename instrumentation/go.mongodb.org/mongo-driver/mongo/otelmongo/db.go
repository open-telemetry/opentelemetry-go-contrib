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

import "go.opentelemetry.io/otel/label"

const (
	DBApplicationKey = label.Key("db.application")
	DBNameKey        = label.Key("db.name")
	DBSystemKey      = label.Key("db.system")
	DBInstanceKey    = label.Key("db.instance")
	DBUserKey        = label.Key("db.user")
	DBStatementKey   = label.Key("db.statement")
)

// DBApplication indicates the application using the database.
func DBApplication(dbApplication string) label.KeyValue {
	return DBApplicationKey.String(dbApplication)
}

// DBName indicates the database name.
func DBName(dbName string) label.KeyValue {
	return DBNameKey.String(dbName)
}

// DBSystem indicates the system of Database.
func DBSystem(dbType string) label.KeyValue {
	return DBSystemKey.String(dbType)
}

// DBInstance indicates the instance name of Database.
func DBInstance(dbInstance string) label.KeyValue {
	return DBInstanceKey.String(dbInstance)
}

// DBUser indicates the user name of Database, e.g. "readonly_user" or "reporting_user".
func DBUser(dbUser string) label.KeyValue {
	return DBUserKey.String(dbUser)
}

// DBStatement records a database statement for the given database type.
func DBStatement(dbStatement string) label.KeyValue {
	return DBStatementKey.String(dbStatement)
}
