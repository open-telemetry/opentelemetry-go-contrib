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

package exponent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type expectMapping struct {
	value float64
	index int64
}

func TestExponentMappingZero(t *testing.T) {
	m := NewExponentMapping(0)

	for _, pair := range []expectMapping{
		{4, 2},
		{3, 1},
		{2, 1},
		{1.5, 0},
		{1, 0},
		{0.75, -1},
		{0.5, -1},
		{0.25, -2},
	} {
		require.Equal(t, pair.index, m.MapToIndex(pair.value))
	}
}

func TestExponentMappingNegOne(t *testing.T) {
	m := NewExponentMapping(-1)

	for _, pair := range []expectMapping{
		{16, 2},
		{15, 1},
		{9, 1},
		{8, 1},
		{5, 1},
		{4, 1},
		{3, 0},
		{2, 0},
		{1.5, 0},
		{1, 0},
		{0.75, -1},
		{0.5, -1},
		{0.25, -1},
		{0.20, -2},
		{0.13, -2},
		{0.125, -2},
		{0.10, -2},
		{0.0625, -2},
		{0.06, -3},
	} {
		require.Equal(t, pair.index, m.MapToIndex(pair.value), "value: %v", pair.value)
	}
}

func TestExponentMappingNegFour(t *testing.T) {
	m := NewExponentMapping(-4)

	for _, pair := range []expectMapping{
		// {float64(0x1), 0},
		// {float64(0x10), 0},
		// {float64(0x100), 0},
		// {float64(0x1000), 0},
		// {float64(0x10000), 1}, // Base == 2**16
		// {float64(0x100000), 1},
		// {float64(0x1000000), 1},
		// {float64(0x10000000), 1},
		// {float64(0x100000000), 2}, // == 2**32
		// {float64(0x1000000000), 2},
		// {float64(0x10000000000), 2},
		// {float64(0x100000000000), 2},
		// {float64(0x1000000000000), 3}, // 2**48
		// {float64(0x10000000000000), 3},
		// {float64(0x100000000000000), 3},
		// {float64(0x1000000000000000), 3},
		// {float64(0x10000000000000000), 4}, // 2**64
		// {float64(0x100000000000000000), 4},
		// {float64(0x1000000000000000000), 4},
		// {float64(0x10000000000000000000), 4},
		// {float64(0x100000000000000000000), 5},

		// {1 / float64(0x1), 0},
		// {1 / float64(0x10), -1},
		// {1 / float64(0x100), -1},
		// {1 / float64(0x1000), -1},
		// {1 / float64(0x10000), -1}, // 2**-16
		// {1 / float64(0x100000), -2},
		// {1 / float64(0x1000000), -2},
		// {1 / float64(0x10000000), -2},
		// {1 / float64(0x100000000), -2}, // 2**-32
		// {1 / float64(0x1000000000), -3},
		// {1 / float64(0x10000000000), -3},
		// {1 / float64(0x100000000000), -3},
		// {1 / float64(0x1000000000000), -3}, // 2**-48
		// {1 / float64(0x10000000000000), -4},
		// {1 / float64(0x100000000000000), -4},
		// {1 / float64(0x1000000000000000), -4},
		// {1 / float64(0x10000000000000000), -4}, // 2**-64
		// {1 / float64(0x100000000000000000), -5},

		// // Max values
		// {0x1p1023, 63},
		// {0x1p1019, 63},
		// {0x1p1008, 63},
		// {0x1p1007, 62},
		// {0x1p1000, 62},
		// {0x1p0992, 62},
		// {0x1p0991, 61},

		// Min and subnormal values
		{0x1p-1074, -68},
		{0x1p-1073, -68},

		// {0x1p-1072, -67},
		// {0x1p-1057, -67},
		// {0x1p-1056, -66},
		// {0x1p-1041, -66},
		// {0x1p-1040, -65},
		// {0x1p-1025, -65},
		// {0x1p-1024, -64},
		// {0x1p-1009, -64},
		// {0x1p-1008, -63},
		// {0x1p-0993, -63},
		// {0x1p-0992, -62},
		// {0x1p-0977, -62},
		// {0x1p-0976, -61},
	} {
		index := m.MapToIndex(pair.value)
		require.Equal(t, pair.index, index, "value: %#x", pair.value)

		lb := m.LowerBoundary(index)
		ub := m.LowerBoundary(index + 1)
		fmt.Println("value/index/lb/ub", pair.value, index, lb, ub)
		require.NotEqual(t, 0., lb)
		require.NotEqual(t, 0., ub)
		require.LessOrEqual(t, lb, pair.value, fmt.Sprintf("value: %x index %v", pair.value, index))
		require.Greater(t, ub, pair.value, fmt.Sprintf("value: %x index %v", pair.value, index))
	}
}
