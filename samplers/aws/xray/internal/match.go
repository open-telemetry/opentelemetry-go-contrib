// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/contrib/samplers/aws/xray/internal"

import (
	"fmt"
	"regexp"
	"strings"
)

// wildcardMatch returns true if text matches pattern at the given case-sensitivity; returns false otherwise.
func wildcardMatch(pattern, text string) (bool, error) {
	patternLen := len(pattern)
	textLen := len(text)
	if patternLen == 0 {
		return textLen == 0, nil
	}

	if pattern == "*" {
		return true, nil
	}

	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	match, err := regexp.MatchString(toRegexPattern(pattern), text)
	if err != nil {
		return false, fmt.Errorf("wildcardMatch: unable to perform regex matching: %w", err)
	}

	return match, nil
}

func toRegexPattern(pattern string) string {
	tokenStart := -1
	var result strings.Builder
	for i, char := range pattern {
		if string(char) == "*" || string(char) == "?" {
			if tokenStart != -1 {
				_, _ = result.WriteString(regexp.QuoteMeta(pattern[tokenStart:i]))
				tokenStart = -1
			}

			if string(char) == "*" {
				_, _ = result.WriteString(".*")
			} else {
				_, _ = result.WriteString(".")
			}
		} else {
			if tokenStart == -1 {
				tokenStart = i
			}
		}
	}
	if tokenStart != -1 {
		_, _ = result.WriteString(regexp.QuoteMeta(pattern[tokenStart:]))
	}
	return result.String()
}
