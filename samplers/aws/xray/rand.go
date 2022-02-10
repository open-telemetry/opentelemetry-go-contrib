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

package xray

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

var _ rand.Source = (*lockedSource)(nil)
var _ rand.Source64 = (*lockedSource64)(nil)

type lockedSource struct {
	mu  sync.Mutex
	src rand.Source
}

func (src *lockedSource) Int63() int64 {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.src.Int63()
}

func (src *lockedSource) Seed(seed int64) {
	src.mu.Lock()
	defer src.mu.Unlock()
	src.src.Seed(seed)
}

type lockedSource64 struct {
	mu  sync.Mutex
	src rand.Source64
}

func (src *lockedSource64) Int63() int64 {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.src.Int63()
}

func (src *lockedSource64) Uint64() uint64 {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.src.Uint64()
}

func (src *lockedSource64) Seed(seed int64) {
	src.mu.Lock()
	defer src.mu.Unlock()
	src.src.Seed(seed)
}

func newSeed() int64 {
	var seed int64
	if err := binary.Read(crand.Reader, binary.BigEndian, &seed); err != nil {
		// fallback to timestamp
		seed = time.Now().UnixNano()
	}
	return seed
}

func newGlobalRand() *rand.Rand {
	src := rand.NewSource(newSeed())
	if src64, ok := src.(rand.Source64); ok {
		return rand.New(&lockedSource64{src: src64})
	}
	return rand.New(&lockedSource{src: src})
}

// Rand is an interface for a set of methods that return random value.
type Rand interface {
	Int63n(n int64) int64
	Intn(n int) int
	Float64() float64
}

// DefaultRand is an implementation of Rand interface.
// It is safe for concurrent use by multiple goroutines.
type defaultRand struct{}

var globalRand = newGlobalRand()

// Int63n returns, as an int64, a non-negative pseudo-random number in [0,n)
// from the default Source.
func (r *defaultRand) Int63n(n int64) int64 {
	return globalRand.Int63n(n)
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n)
// from the default Source.
func (r *defaultRand) Intn(n int) int {
	return globalRand.Intn(n)
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0)
// from the default Source.
func (r *defaultRand) Float64() float64 {
	return globalRand.Float64()
}
