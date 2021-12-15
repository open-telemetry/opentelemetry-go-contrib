// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// Package pattern provides a basic pattern matching utility.
// Patterns may contain fixed text, and/or special characters (`*`, `?`).
// `*` represents 0 or more wildcard characters. `?` represents a single wildcard character.
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

