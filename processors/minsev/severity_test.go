// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"sync"
	"testing"

	"go.opentelemetry.io/otel/log"
)

func TestSeverityVarConcurrentSafe(t *testing.T) {
	var (
		sev SeverityVar
		wg  sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for s := SeverityTrace1; s <= SeverityFatal4; s++ {
			sev.Set(s)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var got log.Severity
		for i := SeverityFatal4 - SeverityTrace1; i >= 0; i-- {
			got = sev.Severity()
		}
		_ = got
	}()

	wg.Wait()
}
