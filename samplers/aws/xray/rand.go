// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray // import "go.opentelemetry.io/contrib/samplers/aws/xray"

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"time"
)

func newSeed() int64 {
	var seed int64
	if err := binary.Read(crand.Reader, binary.BigEndian, &seed); err != nil {
		// fallback to timestamp
		seed = time.Now().UnixNano()
	}
	return seed
}

var seed = newSeed()

func newGlobalRand() *rand.Rand {
	src := rand.NewSource(seed)
	if src64, ok := src.(rand.Source64); ok {
		return rand.New(src64) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
	}
	return rand.New(src) //nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand) is ignored as this is not security-sensitive.
}
