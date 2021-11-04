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
	"database/sql"
	"sync"
	"time"
)

// RecordStats records database statistics for provided sql.DB at the provided
// interval.
func RecordStats(db *sql.DB, interval time.Duration) (fnStop func()) {
	var (
		closeOnce sync.Once
		ctx       = context.Background()
		ticker    = time.NewTicker(interval)
		done      = make(chan struct{})
	)

	go func() {
		for {
			select {
			case <-ticker.C:
				dbStats := db.Stats()
				MeasureOpenConnections.Record(ctx, int64(dbStats.OpenConnections))
				MeasureIdleConnections.Record(ctx, int64(dbStats.Idle))
				MeasureActiveConnections.Record(ctx, int64(dbStats.InUse))
				MeasureWaitCount.Record(ctx, dbStats.WaitCount)
				MeasureWaitDuration.Record(ctx, dbStats.WaitDuration.Milliseconds())
				MeasureIdleClosed.Record(ctx, dbStats.MaxIdleClosed)
				MeasureLifetimeClosed.Record(ctx, dbStats.MaxLifetimeClosed)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		closeOnce.Do(func() {
			close(done)
		})
	}
}
