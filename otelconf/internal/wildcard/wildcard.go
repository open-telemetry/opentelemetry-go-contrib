// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package wildcard matches strings against case-sensitive wildcard patterns.
package wildcard // import "go.opentelemetry.io/contrib/otelconf/internal/wildcard"

import (
	"strings"
	"unicode/utf8"
)

// Matcher matches strings against included and excluded wildcard patterns.
// Patterns in each set are combined with a logical OR, and exclusions take
// precedence over inclusions. An asterisk matches zero or more characters and
// a question mark matches exactly one character.
type Matcher struct {
	included         patternSet
	excluded         patternSet
	includeAll       bool
	excludeAll       bool
	excludeNone      bool
	includeExactOnly bool
}

type patternSet struct {
	exact     map[string]struct{}
	wildcards []string
	matchAll  bool
}

// NewMatcher returns a Matcher for the included and excluded patterns.
func NewMatcher(included, excluded []string) Matcher {
	includedSet := newPatternSet(included)
	excludedSet := newPatternSet(excluded)
	return Matcher{
		included:    includedSet,
		excluded:    excludedSet,
		includeAll:  len(included) == 0 || includedSet.matchAll,
		excludeAll:  excludedSet.matchAll,
		excludeNone: len(excluded) == 0,
		includeExactOnly: len(included) > 0 && len(includedSet.wildcards) == 0 &&
			!includedSet.matchAll && len(excluded) == 0,
	}
}

func newPatternSet(patterns []string) patternSet {
	set := patternSet{}
	for _, pattern := range patterns {
		if pattern == "*" {
			set.matchAll = true
			continue
		}
		if strings.ContainsAny(pattern, "*?") {
			set.wildcards = append(set.wildcards, pattern)
			continue
		}

		if set.exact == nil {
			set.exact = make(map[string]struct{}, len(patterns))
		}
		set.exact[pattern] = struct{}{}
	}
	return set
}

// Match reports whether value is included and not excluded.
func (m Matcher) Match(value string) bool {
	if m.includeExactOnly {
		_, ok := m.included.exact[value]
		return ok
	}
	return m.match(value)
}

func (m Matcher) match(value string) bool {
	if !m.includeAll {
		if _, ok := m.included.exact[value]; !ok && !matchAny(m.included.wildcards, value) {
			return false
		}
	}
	if m.excludeNone {
		return true
	}
	if m.excludeAll {
		return false
	}
	if _, ok := m.excluded.exact[value]; ok {
		return false
	}
	return !matchAny(m.excluded.wildcards, value)
}

func matchAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if match(pattern, value) {
			return true
		}
	}
	return false
}

func match(pattern, value string) bool {
	patternIndex := 0
	valueIndex := 0
	starIndex := -1
	starValueIndex := 0

	for valueIndex < len(value) {
		if patternIndex < len(pattern) {
			switch pattern[patternIndex] {
			case '?':
				_, size := utf8.DecodeRuneInString(value[valueIndex:])
				patternIndex++
				valueIndex += size
				continue
			case '*':
				starIndex = patternIndex
				starValueIndex = valueIndex
				patternIndex++
				continue
			default:
				_, patternSize := utf8.DecodeRuneInString(pattern[patternIndex:])
				_, valueSize := utf8.DecodeRuneInString(value[valueIndex:])
				if patternSize == valueSize &&
					pattern[patternIndex:patternIndex+patternSize] == value[valueIndex:valueIndex+valueSize] {
					patternIndex += patternSize
					valueIndex += valueSize
					continue
				}
			}
		}

		if starIndex < 0 {
			return false
		}
		_, size := utf8.DecodeRuneInString(value[starValueIndex:])
		starValueIndex += size
		patternIndex = starIndex + 1
		valueIndex = starValueIndex
	}

	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}
