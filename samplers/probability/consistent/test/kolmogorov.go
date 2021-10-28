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

package test

import (
	"math"
)

/*
  Copied from the C program to compute Kolmogorov's distribution

  K(n,d) = Prob(D_n < d)

  where

  D_n = max(x_1-0/n,x_2-1/n...,x_n-(n-1)/n,1/n-x_1,2/n-x_2,...,n/n-x_n)

  with  x_1<x_2,...<x_n  a purported set of n independent uniform [0,1)
  random variables sorted into increasing order.

  See G. Marsaglia, Wai Wan Tsang and Jingbo Wong, Journal of
  Statistical Software, 2003.
  https://www.jstatsoft.org/article/view/v008i18
*/
func kolmogorov(n int, d float64) float64 {
	// Omit the next two statements if you require >7 digit accuracy in the right tail.
	s := d * d * float64(n)
	if s > 7.24 || (s > 3.76 && n > 99) {
		return 1 - 2*math.Exp(-(2.000071+.331/math.Sqrt(float64(n))+1.409/float64(n))*s)
	}

	k := int((float64(n) * d) + 1)
	m := 2*k - 1
	h := float64(k) - float64(n)*d
	H := make([]float64, m*m)
	Q := make([]float64, m*m)
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			if i-j+1 < 0 {
				H[i*m+j] = 0
			} else {
				H[i*m+j] = 1
			}
		}
	}
	for i := 0; i < m; i++ {
		H[i*m] -= math.Pow(h, float64(i+1))
		H[(m-1)*m+i] -= math.Pow(h, float64(m-i))
	}
	if 2*h-1 > 0 {
		H[(m-1)*m] += math.Pow(2*h-1, float64(m))
	}
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			if i-j+1 > 0 {
				for g := 1; g <= i-j+1; g++ {
					H[i*m+j] /= float64(g)
				}
			}
		}
	}
	eQ := 0
	mPower(H, 0, Q, &eQ, m, n)
	s = Q[(k-1)*m+k-1]
	for i := 1; i <= n; i++ {
		s = s * float64(i) / float64(n)
		if s < 1e-140 {
			s *= 1e140
			eQ -= 140
		}
	}
	s *= math.Pow(10, float64(eQ))
	return s
}

func mMultiply(A, B, C []float64, m int) {
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			s := 0.0
			for k := 0; k < m; k++ {
				s += A[i*m+k] * B[k*m+j]
			}
			C[i*m+j] = s
		}
	}
}

func mPower(A []float64, eA int, V []float64, eV *int, m, n int) {
	if n == 1 {
		for i := 0; i < m*m; i++ {
			V[i] = A[i]
		}
		*eV = eA
		return
	}
	mPower(A, eA, V, eV, m, n/2)
	B := make([]float64, m*m)
	mMultiply(V, V, B, m)
	eB := 2 * (*eV)
	if n%2 == 0 {
		for i := 0; i < m*m; i++ {
			V[i] = B[i]
			*eV = eB
		}
	} else {
		mMultiply(A, B, V, m)
		*eV = eA + eB
	}
	if V[(m/2)*m+(m/2)] > 1e140 {
		for i := 0; i < m*m; i++ {
			V[i] = V[i] * 1e-140
		}
		*eV += 140
	}
}
