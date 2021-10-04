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
	"fmt"
	"math/rand"
	"testing"

	histogram "github.com/jmacd/otlp-expo-histo"
)

func benchmarkHistogram(b *testing.B, name string, mapper histogram.Base2HistogramMapper, scale int) {
	b.Run(fmt.Sprintf("mapping_%s_%d", name, scale), func(b *testing.B) {
		src := rand.New(rand.NewSource(54979))

		for i := 0; i < b.N; i++ {
			_ = mapper.MapToIndex(1 + src.Float64())
		}
	})
	b.Run(fmt.Sprintf("boundary_%s_%d", name, scale), func(b *testing.B) {
		src := rand.New(rand.NewSource(54979))

		for i := 0; i < b.N; i++ {
			_ = mapper.LowerBoundary(src.Int63())
		}
	})
}

func BenchmarkHistogram(b *testing.B) {
	for _, scale := range []int{3, 10} {
		benchmarkHistogram(b, "lookup", histogram.NewLookupTableMapping(scale), scale)
		benchmarkHistogram(b, "logarithm", histogram.NewLogarithmMapping(scale), scale)
	}
}
