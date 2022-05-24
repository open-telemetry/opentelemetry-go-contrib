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
