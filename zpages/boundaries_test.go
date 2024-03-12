// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testDurations = []time.Duration{1 * time.Second}

func TestBoundariesNumBuckets(t *testing.T) {
	assert.Equal(t, 1, newBoundaries(nil).numBuckets())
	assert.Equal(t, 1, newBoundaries([]time.Duration{}).numBuckets())
	assert.Equal(t, 2, newBoundaries(testDurations).numBuckets())
	assert.Equal(t, 9, defaultBoundaries.numBuckets())
}

func TestBoundariesGetBucketIndex(t *testing.T) {
	assert.Equal(t, 0, newBoundaries(testDurations).getBucketIndex(zeroDuration))
	assert.Equal(t, 0, newBoundaries(testDurations).getBucketIndex(500*time.Millisecond))
	assert.Equal(t, 1, newBoundaries(testDurations).getBucketIndex(1500*time.Millisecond))
	assert.Equal(t, 0, newBoundaries(testDurations).getBucketIndex(zeroDuration))

	assert.Equal(t, 0, defaultBoundaries.getBucketIndex(zeroDuration))
	assert.Equal(t, 3, defaultBoundaries.getBucketIndex(5*time.Millisecond))
	assert.Equal(t, 6, defaultBoundaries.getBucketIndex(5*time.Second))
	assert.Equal(t, 8, defaultBoundaries.getBucketIndex(maxDuration))
}
