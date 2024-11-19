// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consistent // import "go.opentelemetry.io/contrib/samplers/probability/consistent"

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	traceStateKey       = "ot"
	pValueSubkey        = "p"
	rValueSubkey        = "r"
	pZeroValue          = 63
	invalidValue        = pZeroValue + 1 // invalid for p or r
	traceStateSizeLimit = 256
)

var (
	errTraceStateSyntax       = fmt.Errorf("otel tracestate: %w", strconv.ErrSyntax)
	errTraceStateInconsistent = errors.New("r-value and p-value are inconsistent")
)

type otelTraceState struct {
	rvalue  uint8 // valid in the interval [0, 62]
	pvalue  uint8 // valid in the interval [0, 63]
	unknown []string
}

func newTraceState() otelTraceState {
	return otelTraceState{
		rvalue: invalidValue, // out-of-range => !hasRValue()
		pvalue: invalidValue, // out-of-range => !hasPValue()
	}
}

func (otts otelTraceState) serialize() string {
	var sb strings.Builder
	semi := func() {
		if sb.Len() != 0 {
			_, _ = sb.WriteString(";")
		}
	}

	if otts.hasPValue() {
		_, _ = sb.WriteString(fmt.Sprintf("p:%d", otts.pvalue))
	}
	if otts.hasRValue() {
		semi()
		_, _ = sb.WriteString(fmt.Sprintf("r:%d", otts.rvalue))
	}
	for _, unk := range otts.unknown {
		ex := 0
		if sb.Len() != 0 {
			ex = 1
		}
		if sb.Len()+ex+len(unk) > traceStateSizeLimit {
			// Note: should this generate an explicit error?
			break
		}
		semi()
		_, _ = sb.WriteString(unk)
	}
	return sb.String()
}

func isValueByte(r byte) bool {
	if isLCAlphaNum(r) {
		return true
	}
	if isUCAlpha(r) {
		return true
	}
	return r == '.' || r == '_' || r == '-'
}

func isLCAlphaNum(r byte) bool {
	if isLCAlpha(r) {
		return true
	}
	return r >= '0' && r <= '9'
}

func isLCAlpha(r byte) bool {
	return r >= 'a' && r <= 'z'
}

func isUCAlpha(r byte) bool {
	return r >= 'A' && r <= 'Z'
}

func parseOTelTraceState(ts string, isSampled bool) (otelTraceState, error) { // nolint: revive
	var pval, rval string
	var unknown []string

	if len(ts) == 0 {
		return newTraceState(), nil
	}

	if len(ts) > traceStateSizeLimit {
		return newTraceState(), errTraceStateSyntax
	}

	for len(ts) > 0 {
		eqPos := 0
		for ; eqPos < len(ts); eqPos++ {
			if eqPos == 0 {
				if isLCAlpha(ts[eqPos]) {
					continue
				}
			} else if isLCAlphaNum(ts[eqPos]) {
				continue
			}
			break
		}
		if eqPos == 0 || eqPos == len(ts) || ts[eqPos] != ':' {
			return newTraceState(), errTraceStateSyntax
		}

		key := ts[0:eqPos]
		tail := ts[eqPos+1:]

		sepPos := 0

		for ; sepPos < len(tail); sepPos++ {
			if isValueByte(tail[sepPos]) {
				continue
			}
			break
		}

		if key == pValueSubkey {
			// Note: does the spec say how to handle duplicates?
			pval = tail[0:sepPos]
		} else if key == rValueSubkey {
			rval = tail[0:sepPos]
		} else {
			unknown = append(unknown, ts[0:sepPos+eqPos+1])
		}

		if sepPos < len(tail) && tail[sepPos] != ';' {
			return newTraceState(), errTraceStateSyntax
		}

		if sepPos == len(tail) {
			break
		}

		ts = tail[sepPos+1:]

		// test for a trailing ;
		if ts == "" {
			return newTraceState(), errTraceStateSyntax
		}
	}

	otts := newTraceState()
	otts.unknown = unknown

	// Note: set R before P, so that P won't propagate if R has an error.
	value, err := parseNumber(rValueSubkey, rval, pZeroValue-1)
	if err != nil {
		return otts, err
	}
	otts.rvalue = value

	value, err = parseNumber(pValueSubkey, pval, pZeroValue)
	if err != nil {
		return otts, err
	}
	otts.pvalue = value

	// Invariant checking: unset P when the values are inconsistent.
	if otts.hasPValue() && otts.hasRValue() {
		implied := otts.pvalue <= otts.rvalue || otts.pvalue == pZeroValue

		if !isSampled || !implied {
			// Note: the error ensures the parent-based
			// sampler repairs the broken tracestate entry.
			otts.pvalue = invalidValue
			return otts, parseError(pValueSubkey, errTraceStateInconsistent)
		}
	}

	return otts, nil
}

func parseNumber(key string, input string, maximum uint8) (uint8, error) {
	if input == "" {
		return maximum + 1, nil
	}
	value, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return maximum + 1, parseError(key, err)
	}
	if value > uint64(maximum) {
		return maximum + 1, parseError(key, strconv.ErrRange)
	}
	// `value` is strictly less then the uint8 maximum. This cast is safe.
	return uint8(value), nil // nolint: gosec
}

func parseError(key string, err error) error {
	return fmt.Errorf("otel tracestate: %s-value %w", key, err)
}

func (otts otelTraceState) hasRValue() bool {
	return otts.rvalue < pZeroValue
}

func (otts otelTraceState) hasPValue() bool {
	return otts.pvalue <= pZeroValue
}
