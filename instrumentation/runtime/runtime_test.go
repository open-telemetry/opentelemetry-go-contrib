// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime // import "go.opentelemetry.io/contrib/instrumentation/runtime"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefreshGoCollector(t *testing.T) {
	// buffer for allocating memory
	var buffer [][]byte
	collector := newCollector(10 * time.Second)
	testClock := newClock()
	collector.now = testClock.now
	// before the first refresh, all counters are zero
	assert.Zero(t, collector.get(goMemoryAllocations))
	// after the first refresh, counters are non-zero
	collector.refresh()
	initialAllocations := collector.get(goMemoryAllocations)
	assert.NotZero(t, initialAllocations)
	// if less than the refresh time has elapsed, the value is not updated
	// on refresh.
	testClock.increment(9 * time.Second)
	collector.refresh()
	allocateMemory(buffer)
	assert.Equal(t, initialAllocations, collector.get(goMemoryAllocations))
	// if greater than the refresh time has elapsed, the value changes.
	testClock.increment(2 * time.Second)
	collector.refresh()
	allocateMemory(buffer)
	assert.NotEqual(t, initialAllocations, collector.get(goMemoryAllocations))
}

func allocateMemory(buffer [][]byte) [][]byte {
	newBuffer := make([]byte, 100000)
	for i := range newBuffer {
		newBuffer[i] = 0
	}
	buffer = append(buffer, newBuffer)
	return buffer
}

func newClock() *clock {
	return &clock{current: time.Now()}
}

type clock struct {
	current time.Time
}

func (c *clock) now() time.Time { return c.current }

func (c *clock) increment(d time.Duration) { c.current = c.current.Add(d) }
