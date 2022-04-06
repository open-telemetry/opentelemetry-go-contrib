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

package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	"time"
)

// ticker is the same as time.Ticker except that it has jitters.
// A Ticker must be created with newTicker.
type ticker struct {
	tick     *time.Ticker
	duration time.Duration
	jitter   time.Duration
}

// newTicker creates a new Ticker that will send the current time on its channel with the passed jitter.
func newTicker(duration, jitter time.Duration) *ticker {
	t := time.NewTicker(duration - time.Duration(newGlobalRand().Int63n(int64(jitter))))

	jitteredTicker := ticker{
		tick:     t,
		duration: duration,
		jitter:   jitter,
	}

	return &jitteredTicker
}

// c returns a channel that receives when the ticker fires.
func (j *ticker) c() <-chan time.Time {
	return j.tick.C
}
