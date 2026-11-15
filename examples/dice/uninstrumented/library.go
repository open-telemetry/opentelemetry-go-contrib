// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"math/rand/v2"
)

func rolldice(rolls int) ([]int, error) {
	const maxRolls = 1000 // Arbitrary limit to prevent Slice memory allocation with excessive size value.

	if rolls > maxRolls {
		return nil, errors.New("rolls parameter exceeds maximum allowed value")
	}

	if rolls <= 0 {
		return nil, errors.New("rolls must be positive")
	}

	if rolls == 1 {
		return []int{rollOnce()}, nil
	}

	results := make([]int, rolls)
	for i := range rolls {
		results[i] = rollOnce()
	}
	return results, nil
}

// rollOnce returns a random number between 1 and 6.
func rollOnce() int {
	roll := 1 + rand.IntN(6) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	return roll
}
