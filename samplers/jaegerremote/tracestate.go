// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote // import "go.opentelemetry.io/contrib/samplers/jaegerremote"

import (
	"strconv"
	"strings"
)

// insertOrUpdateTraceStateThKeyValue inserts or updates the threshold (th)
// key-value in the OpenTelemetry tracestate "ot" field, preserving any other
// key-values already present.
func insertOrUpdateTraceStateThKeyValue(existingOtts, thkv string) string {
	if existingOtts == "" {
		return thkv
	}

	start := -1
	var end int
	if strings.HasPrefix(existingOtts, "th:") {
		start = 0
	} else if idx := strings.Index(existingOtts, ";th:"); idx != -1 {
		start = idx + 1
	}
	if start == -1 {
		return thkv + ";" + existingOtts
	}

	for end = start; end < len(existingOtts); end++ {
		if existingOtts[end] == ';' {
			end++
			break
		}
	}

	if end == len(existingOtts) {
		return strings.TrimSuffix(thkv+";"+existingOtts[0:start], ";")
	}
	return thkv + ";" + existingOtts[0:start] + existingOtts[end:]
}

// tracestateRandomness determines whether there is a randomness "rv"
// sub-key in otts (the top-level OpenTelemetry tracestate "ot" field). If
// present, "rv" is a 56-bit unsigned integer, encoded in 14 hex digits.
func tracestateRandomness(otts string) (randomness uint64, hasRandomness bool) {
	var start int
	if strings.HasPrefix(otts, "rv:") {
		start = 3
	} else if idx := strings.Index(otts, ";rv:"); idx != -1 {
		start = idx + 4
	} else {
		return 0, false
	}

	if len(otts) < start+14 || (len(otts) > start+14 && otts[start+14] != ';') {
		return 0, false
	}

	rv, err := strconv.ParseUint(otts[start:start+14], 16, 56)
	if err != nil {
		return 0, false
	}
	return rv, true
}
