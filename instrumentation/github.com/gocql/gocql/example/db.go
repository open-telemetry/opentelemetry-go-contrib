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

package main

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
)

func initDb() {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "system"
	cluster.Consistency = gocql.LocalQuorum
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	stmt := fmt.Sprintf(
		"create keyspace if not exists %s with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
		keyspace,
	)
	if err := session.Query(stmt).Exec(); err != nil {
		log.Fatal(err)
	}

	cluster.Keyspace = keyspace
	session, err = cluster.CreateSession()

	stmt = "create table if not exists book(id UUID, title text, author_first_name text, author_last_name text, PRIMARY KEY(id))"
	if err = session.Query(stmt).Exec(); err != nil {
		log.Fatal(err)
	}

	if err := session.Query("create index if not exists on book(author_last_name)").Exec(); err != nil {
		log.Fatal(err)
	}
}
