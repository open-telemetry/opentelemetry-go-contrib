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

package otelgorm

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go.opentelemetry.io/contrib/internal/util"

	"go.opentelemetry.io/otel/oteltest"
)

type TestModel struct {
	ID    uint `gorm:"primarykey"`
	Code  string
	Price uint
}

func initDB() (*gorm.DB, error) {
	var err error
	var dbFile *os.File
	var db *gorm.DB

	dbFile, err = ioutil.TempFile("", "db")
	defer func() {
		if err == nil {
			return
		}

		if dbFile != nil {
			os.Remove(dbFile.Name())
		}

		if db != nil {
			closeDB(db)
		}

	}()

	if err != nil {
		return nil, err
	}

	// Initialize db connection
	db, err = gorm.Open(sqlite.Open(dbFile.Name()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&TestModel{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func closeDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		sqlDB.Close()
	}
}

func TestPlugin(t *testing.T) {

	testCases := []struct {
		name           string
		testOp         func(db *gorm.DB) *gorm.DB
		spans          int
		targetSpan     int
		expectSpanName string
		sqlOp          string
		affectedRows   int64
	}{
		{
			"create (insert) row",
			func(db *gorm.DB) *gorm.DB {
				return db.Create(&TestModel{Code: "D42", Price: 100})
			},
			2,
			0,
			"INSERT db.test_models",
			"INSERT",
			1,
		},
		{
			"save (update) row",
			func(db *gorm.DB) *gorm.DB {
				tm := TestModel{ID: 1, Code: "D42", Price: 100}
				db = db.Create(&tm)
				if db.Error != nil {
					return db
				}
				tm.Code = "foo"
				return db.Save(&tm)
			},
			3,
			1,
			"UPDATE db.test_models",
			"UPDATE",
			1,
		},
		{
			"delete row",
			func(db *gorm.DB) *gorm.DB {
				tm := TestModel{ID: 1, Code: "D42", Price: 100}
				db = db.Create(&tm)
				if db.Error != nil {
					return db
				}
				return db.Delete(&tm)
			},
			3,
			1,
			"DELETE db.test_models",
			"DELETE",
			1,
		},
		{
			"query row",
			func(db *gorm.DB) *gorm.DB {
				tm := TestModel{ID: 1, Code: "D42", Price: 100}
				db = db.Create(&tm)
				if db.Error != nil {
					return db
				}
				return db.First(&tm)
			},
			3,
			1,
			"SELECT db.test_models",
			"SELECT",
			1,
		},
		{
			"raw",
			func(db *gorm.DB) *gorm.DB {
				tm := TestModel{ID: 1, Code: "D42", Price: 100}
				db = db.Create(&tm)
				if db.Error != nil {
					return db
				}

				var result []TestModel
				return db.Raw("SELECT * FROM test_models").Scan(&result)
			},
			3,
			1,
			"SELECT db.test_models",
			"SELECT",
			-1,
		},
		{
			"row",
			func(db *gorm.DB) *gorm.DB {
				tm := TestModel{ID: 1, Code: "D42", Price: 100}
				db = db.Create(&tm)
				if db.Error != nil {
					return db
				}

				db.Raw("SELECT id FROM test_models").Row()
				return &gorm.DB{Error: nil}
			},
			3,
			1,
			"SELECT db.test_models",
			"SELECT",
			-1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			db, err := initDB()
			defer closeDB(db)

			assert.NoError(tt, err)

			sr := new(oteltest.StandardSpanRecorder)
			provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))

			plugin := NewPlugin(WithTracerProvider(provider), WithDBName("db"))

			err = db.Use(plugin)
			assert.NoError(tt, err)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			ctx, span := provider.Tracer(defaultTracerName).Start(ctx, "gorm-test")

			db = db.WithContext(ctx)
			// Create
			dbOp := tc.testOp(db)
			assert.NoError(tt, dbOp.Error)

			span.End()

			spans := sr.Completed()
			require.Len(t, spans, tc.spans)
			s := spans[tc.targetSpan]

			assert.Equal(tt, spans[0].SpanContext().TraceID, spans[1].SpanContext().TraceID)
			assert.Equal(tt, s.Name(), tc.expectSpanName)
			assert.Equal(tt, "test_models", s.Attributes()[dbTableKey].AsString())
			assert.Equal(tt, tc.sqlOp, s.Attributes()[dbOperationKey].AsString())
			assert.Equal(tt, tc.affectedRows, s.Attributes()[dbRowsAffectedKey].AsInt64())
			assert.Contains(tt, s.Attributes()[dbStatementKey].AsString(), tc.sqlOp)
		})
	}

}

func TestMain(m *testing.M) {
	util.IntegrationShouldRun("test-gorm")
	os.Exit(m.Run())
}
