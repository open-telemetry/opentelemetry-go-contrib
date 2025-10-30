// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"math/rand"
)

// rolldice is the outer function which does the error handling.
func rolldice(rolls int) ([]int, error) {
	if rolls <= 0 {
		return nil, errors.New("rolls must be positive")
	}

	if rolls == 1 {
		return []int{rollOnce()}, nil
	}

	results := make([]int, rolls)
	for i := 0; i < rolls; i++ {
		results[i] = rollOnce()
	}
	return results, nil
}

// rollOnce is the inner function — returns a random number 1–6.
func rollOnce() int {
	roll := 1 + rand.Intn(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	return roll
}
