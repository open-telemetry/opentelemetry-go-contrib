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

package zpages

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type testSpan struct {
	sdktrace.ReadWriteSpan
	spanContext trace.SpanContext
	name        string
	startTime   time.Time
	endTime     time.Time
	status      sdktrace.Status
}

func (ts *testSpan) SpanContext() trace.SpanContext {
	return ts.spanContext
}

func (ts *testSpan) Status() sdktrace.Status {
	return ts.status
}

func (ts *testSpan) Name() string {
	return ts.name
}

func (ts *testSpan) StartTime() time.Time {
	return ts.startTime
}

func (ts *testSpan) EndTime() time.Time {
	return ts.endTime
}

func TestBucket(t *testing.T) {
	bkt := newBucket(defaultBucketCapacity)
	assert.Equal(t, 0, bkt.len())

	for i := 1; i <= defaultBucketCapacity; i++ {
		bkt.add(&testSpan{endTime: time.Unix(int64(i), 0)})
		assert.Equal(t, i, bkt.len())
		spans := bkt.spans()
		assert.Len(t, spans, i)
		for j := 0; j < i; j++ {
			assert.Equal(t, time.Unix(int64(j+1), 0), spans[j].EndTime())
		}
	}

	for i := defaultBucketCapacity + 1; i <= 2*defaultBucketCapacity; i++ {
		bkt.add(&testSpan{endTime: time.Unix(int64(i), 0)})
		assert.Equal(t, defaultBucketCapacity, bkt.len())
		spans := bkt.spans()
		assert.Len(t, spans, defaultBucketCapacity)
		// First spans will have newer times, and will replace older timestamps.
		for j := 0; j < i-defaultBucketCapacity; j++ {
			assert.Equal(t, time.Unix(int64(j+defaultBucketCapacity+1), 0), spans[j].EndTime())
		}
		for j := i - defaultBucketCapacity; j < defaultBucketCapacity; j++ {
			assert.Equal(t, time.Unix(int64(j+1), 0), spans[j].EndTime())
		}
	}
}

func TestBucketAddSample(t *testing.T) {
	bkt := newBucket(defaultBucketCapacity)
	assert.Equal(t, 0, bkt.len())

	for i := 0; i < 1000; i++ {
		bkt.add(&testSpan{endTime: time.Unix(1, int64(i*1000))})
		assert.Equal(t, 1, bkt.len())
		spans := bkt.spans()
		assert.Len(t, spans, 1)
		assert.Equal(t, time.Unix(1, 0), spans[0].EndTime())
	}
}

func TestBucketZeroCapacity(t *testing.T) {
	bkt := newBucket(0)
	assert.Equal(t, 0, bkt.len())
	bkt.add(&testSpan{endTime: time.Unix(1, 0)})
	assert.Equal(t, 0, bkt.len())
	assert.Len(t, bkt.spans(), 0)
}
