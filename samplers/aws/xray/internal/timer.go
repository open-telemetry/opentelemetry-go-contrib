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

// Ticker is the same as time.Ticker except that it has jitters.
// A Ticker must be created with NewTicker.
type Ticker struct {
	Tick     *time.Ticker
	duration time.Duration
	jitter   time.Duration
}

// NewTicker creates a new Ticker that will send the current time on its channel.
func NewTicker(duration, jitter time.Duration) *Ticker {
	t := time.NewTicker(duration - time.Duration(newGlobalRand().Int63n(int64(jitter))))

	jitteredTicker := Ticker{
		Tick:     t,
		duration: duration,
		jitter:   jitter,
	}

	return &jitteredTicker
}

// C is channel.
func (j *Ticker) C() <-chan time.Time {
	return j.Tick.C
}
