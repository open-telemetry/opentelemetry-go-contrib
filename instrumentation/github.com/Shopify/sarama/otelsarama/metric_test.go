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

package otelsarama

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO(#2755): Add integration tests for the host instrumentation. These tests
// depend on https://github.com/open-telemetry/opentelemetry-go/issues/3031
// being resolved.
//
// The added tests will depend on the metric SDK. Therefore, they should be
// added to a sub-directory called "test" instead of this file.

func TestRateMetric(t *testing.T) {
	rmetric := newRateMetric()
	assert.NotNil(t, rmetric)

	var wg sync.WaitGroup

	var record float64 = 100
	numAdditions := 10000

	wg.Add(numAdditions)
	for i := 0; i < numAdditions; i++ {
		go func(record float64) {
			rmetric.Add(record)
			wg.Done()
		}(record)
	}
	wg.Wait()

	assert.Equal(t, rmetric.recordAccumulation, record*float64(numAdditions))

	// needs to be positive value.
	avg := rmetric.Average()
	assert.Greater(t, avg, float64(0))

	loadedAfterFlush := rmetric.recordAccumulation
	t.Log(loadedAfterFlush)
	assert.Equal(t, loadedAfterFlush, float64(0))
}
