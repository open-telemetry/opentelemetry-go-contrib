// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package histogram_test

import (
	"math/big"
	"math/rand"
	"testing"

	histogram "github.com/jmacd/otlp-expo-histo"
	"github.com/stretchr/testify/assert"
)

var (
	onef = big.NewFloat(1)
	onei = big.NewInt(1)
)

func newf() *big.Float {
	return &big.Float{}
}

func newi() *big.Int {
	return &big.Int{}
}

func pow2(x int) *big.Float {
	return newf().SetMantExp(onef, x)
}

func toInt64(x *big.Float) *big.Int {
	i, _ := x.SetMode(big.ToZero).Int64()
	return big.NewInt(i)
}

func ipow(b *big.Int, p int64) *big.Int {
	r := onei
	for i := int64(0); i < p; i++ {
		r = newi().Mul(r, b)
	}
	return r
}

func TestBoundariesAreExact(t *testing.T) {
	input := histogram.ExponentialConstants
	size := int64(len(input))

	// Validate 25 random entries.
	for i := 0; i < 25; i++ {
		position := rand.Intn(len(input))
		// x is a 52-bit number representing the mantissa of
		// the normalized floating point value in the range
		// [1,2) that is the base-2 logarithm 2^(position/size).
		x := input[position]

		scaled := newf().Add(big.NewFloat(float64(x)), pow2(52))
		normed := toInt64(scaled) // in the range [2^52, 2^53)
		compareTo, _ := pow2(52*int(size) + position).Int(nil)

		// Test that the mantissa is unchanged:
		assert.Equal(t, x, normed.Uint64()&histogram.MantissaMask)

		// normed^size should be greater or equal to the
		// inclusive lower bound.  Test is (-1 < cmp())
		assert.Less(t, -1, ipow(normed, size).Cmp(compareTo))

		// (normed-1)^size should be less than the inclusive
		// lower bound.  Test is (0 > tmp())
		assert.Greater(t, 0, ipow(newi().Sub(normed, onei), size).Cmp(compareTo))
	}
}
