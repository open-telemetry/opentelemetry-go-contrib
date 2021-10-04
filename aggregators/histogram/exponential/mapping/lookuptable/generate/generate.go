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

package main

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	histogram "github.com/jmacd/otlp-expo-histo"
)

var startTime = time.Now()

func runningTime() time.Duration {
	var usage syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	if err != nil {
		return time.Since(startTime)
	}
	return time.Duration(usage.Utime.Sec+usage.Stime.Sec) * time.Second
}

// main prints a table of constants for use in a lookup-table
// implementation of the base2 exponential histogram of OTEP 149.
//
// Derived from https://github.com/dynatrace-oss/dynahist/commit/abc6ba2e5b49760247591f9597873d24e01c4374
//
// Note: this has exponential time complexity because the number of
// math/big operations per entry in the table is O(2**scale).
//
// "generate 12" takes around 5 minutes of CPU time.
//
// See https://gist.github.com/jmacd/64031dc003410db6eb3ae04a1db286fd
// for the table of scale=16 constants (this took 27 cpu-days).
func main() {
	scale, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("usage: %s scale (an integer)\n", os.Args[1])
		os.Exit(1)
	}

	var (
		// size is 2**scale
		size       = int64(1) << scale
		thresholds = make([]uint64, size)

		// constants
		onef = big.NewFloat(1)
		onei = big.NewInt(1)
	)

	newf := func() *big.Float { return &big.Float{} }
	newi := func() *big.Int { return &big.Int{} }
	pow2 := func(x int) *big.Float {
		return newf().SetMantExp(onef, x)
	}
	toInt64 := func(x *big.Float) *big.Int {
		i, _ := x.SetMode(big.ToZero).Int64()
		return big.NewInt(i)
	}
	ipow := func(b *big.Int, p int64) *big.Int {
		r := onei
		for i := int64(0); i < p; i++ {
			r = newi().Mul(r, b)
		}
		return r
	}

	var finished int64
	var wg sync.WaitGroup

	// Round to a power of two smaller than NumCPU
	ncpu := 1 << (64 - bits.LeadingZeros64(uint64(runtime.NumCPU())))
	percpu := len(thresholds) / ncpu
	wg.Add(ncpu)

	go func() {
		// Since this can take a long time to run for large
		// scales, print a progress report.
		t := time.NewTicker(time.Minute)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if finished == 0 {
					continue
				}
				elapsed := time.Since(startTime)
				cpu := runningTime()
				count := atomic.LoadInt64(&finished)
				os.Stderr.WriteString(fmt.Sprintf("%d @ %s: %.4f%% complete %s remaining...\n",
					count,
					elapsed.Round(time.Minute),
					100*float64(count)/float64(size),
					time.Duration(
						(float64(size)*float64(cpu)/float64(count)-float64(cpu))/float64(runtime.NumCPU()),
					).Round(time.Minute),
				))
			}
		}
	}()

	for cpu := 0; cpu < ncpu; cpu++ {
		go func(cpu int) {
			defer wg.Done()
			for j := 0; j < percpu; j++ {
				position := cpu*percpu + j

				// whereas (position/size) in the range [0, 1),
				//   x = 2**(position/size)
				// falls in the range [1, 2).  Equivalently,
				// calculate 2**position, then square-root scale times.
				x := pow2(position)
				for i := 0; i < scale; i++ {
					x = newf().Sqrt(x)
				}

				// Compute the integer value in the range [2**52, 2**53)
				// which is the 52-bit significand of the IEEE float64
				// as an uint64 value plus 2**52, alternatively the value
				// x with range [1, 2) times 2**52.
				scaled := newf().Mul(x, pow2(52))
				normed := toInt64(scaled) // in the range [2**52, 2**53)
				compareTo, _ := pow2(52*int(size) + position).Int(nil)

				for {
					candidate := ipow(normed, size)
					compare := candidate.Cmp(compareTo)

					if compare == 0 {
						// perfect!
						break
					}

					if compare < 0 {
						normed = newi().Add(normed, onei)
						// This happens frequently.
						continue
					}

					// ensure that subtracting one
					// produces a smaller comparision.
					normedMinus := newi().Sub(normed, onei)
					candidateMinus := ipow(normedMinus, size)
					compareMinus := candidateMinus.Cmp(compareTo)

					// If (normed-1)**size is greater than or equal to the
					// inclusive lower bound
					if compareMinus >= 0 {
						// This happens rarely.  First discovered
						// by running an earlier version of this
						// program with scale=16 at position @ 33311.
						normed = normedMinus
						continue
					} else {
						break
					}
				}

				thresholds[position] = normed.Uint64() & ((uint64(1) << 52) - 1)

				atomic.AddInt64(&finished, 1)
			}
		}(cpu)
	}
	wg.Wait()

	fmt.Printf(`
package histogram

// ExponentialConstants is a table of logarithms, exactly computed
// with 52-bits of precision for use comparing with the significand
// (i.e., mantissa) of an IEEE 754 double-width floating-point value.
// See OpenTelemetry OTEP 149 for details on this histogram.
var ExponentialConstants = [%d]uint64{
`, size)

	for pos, value := range thresholds {
		fmt.Printf("\t0x%012x,  // 2**(%d/%d) == %.016g\n",
			value,
			pos,
			size,
			math.Float64frombits((uint64(histogram.ExponentBias)<<histogram.MantissaWidth)+value),
		)
	}
	fmt.Printf(`}
`)
}
