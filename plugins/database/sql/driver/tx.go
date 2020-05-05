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

package driver

import (
	"context"
	"database/sql/driver"
)

type (
	tx struct {
		t driver.Tx
		s *tracingSetup
	}
)

var _ driver.Tx = &tx{}

func maybeNewTx(realTx driver.Tx, setup *tracingSetup) driver.Tx {
	if realTx == nil {
		return nil
	}
	return newTx(realTx, setup)
}

func newTx(realTx driver.Tx, setup *tracingSetup) driver.Tx {
	return &tx{
		t: realTx,
		s: setup,
	}
}

func (t *tx) Commit() error {
	ctx, span := t.s.tracer.Start(context.Background(), "transaction commit")
	err := t.t.Commit()
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return err
}

func (t *tx) Rollback() error {
	ctx, span := t.s.tracer.Start(context.Background(), "transaction rollback")
	err := t.t.Rollback()
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	span.End()
	return err
}
