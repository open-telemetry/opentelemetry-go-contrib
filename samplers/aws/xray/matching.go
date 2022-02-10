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

package xray

import "strings"

// wildcardMatch returns true if text matches pattern at the given case-sensitivity; returns false otherwise.
func wildcardMatch(pattern, text string, caseInsensitive bool) bool {
	patternLen := len(pattern)
	textLen := len(text)
	if patternLen == 0 {
		return textLen == 0
	}

	if pattern == "*" {
		return true
	}

	if caseInsensitive {
		pattern = strings.ToLower(pattern)
		text = strings.ToLower(text)
	}

	i := 0
	p := 0
	iStar := textLen
	pStar := 0

	for i < textLen {
		if p < patternLen {
			switch pattern[p] {
			case text[i]:
				i++
				p++
				continue
			case '?':
				i++
				p++
				continue
			case '*':
				iStar = i
				pStar = p
				p++
				continue
			}
		}
		if iStar == textLen {
			return false
		}
		iStar++
		i = iStar
		p = pStar + 1
	}

	for p < patternLen && pattern[p] == '*' {
		p++
	}

	return p == patternLen && i == textLen
}
