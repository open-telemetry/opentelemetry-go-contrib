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

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"time"
)

// clock represents a time keeper that returns its version of the current time.
type clock interface {
	now() time.Time
}

// defaultClock wraps the standard time package.
type defaultClock struct{}

// now returns current time according to the standard time package.
func (t *defaultClock) now() time.Time {
	return time.Now()
}

// mockClock is a time keeper that returns a fixed time.
type mockClock struct {
	nowTime  int64
	nowNanos int64
}

// now function returns the fixed time value stored in c.
func (c *mockClock) now() time.Time {
	return time.Unix(c.nowTime, c.nowNanos)
}
