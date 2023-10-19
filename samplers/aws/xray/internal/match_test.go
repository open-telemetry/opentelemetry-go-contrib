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

package internal

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assert wildcard match is positive.
func TestWildCardMatchPositive(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
	}{
		// wildcard positive test set
		{"*", ""},
		{"foo", "foo"},
		{"foo*bar*?", "foodbaris"},
		{"?o?", "foo"},
		{"*oo", "foo"},
		{"foo*", "foo"},
		{"*o?", "foo"},
		{"*", "boo"},
		{"", ""},
		{"a", "a"},
		{"*a", "a"},
		{"*a", "ba"},
		{"a*", "a"},
		{"a*", "ab"},
		{"a*a", "aa"},
		{"a*a", "aba"},
		{"a*a*", "aaaaaaaaaaaaaaaaaaaaaaa"},
		{
			"a*b*a*b*a*b*a*b*a*",
			"akljd9gsdfbkjhaabajkhbbyiaahkjbjhbuykjakjhabkjhbabjhkaabbabbaaakljdfsjklababkjbsdabab",
		},
		{"a*na*ha", "anananahahanahana"},
		{"***a", "a"},
		{"**a**", "a"},
		{"a**b", "ab"},
		{"*?", "a"},
		{"*??", "aa"},
		{"*?", "a"},
		{"*?*a*", "ba"},
		{"?at", "bat"},
		{"?at", "cat"},
		{"?o?se", "horse"},
		{"?o?se", "mouse"},
		{"*s", "horse"},
		{"J*", "Jeep"},
		{"J*", "jeep"},
		{"*/foo", "/bar/foo"},
	}

	for _, test := range tests {
		match, err := wildcardMatch(test.pattern, test.text)
		require.NoError(t, err)
		assert.True(t, match, test.text)
	}
}

// assert wildcard match is negative.
func TestWildCardMatchNegative(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
	}{
		// wildcard negative test set
		{"", "whatever"},
		{"foo", "bar"},
		{"f?o", "boo"},
		{"f??", "boo"},
		{"fo*", "boo"},
		{"f?*", "boo"},
		{"abcd", "abc"},
		{"??", "a"},
		{"??", "a"},
		{"*?*a", "a"},
	}

	for _, test := range tests {
		match, err := wildcardMatch(test.pattern, test.text)
		require.NoError(t, err)
		assert.False(t, match)
	}
}

func TestLongStrings(t *testing.T) {
	chars := []byte{'a', 'b', 'c', 'd'}
	text := bytes.NewBufferString("a")
	for i := 0; i < 8192; i++ {
		_, _ = text.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	_, _ = text.WriteString("b")

	match, err := wildcardMatch("a*b", text.String())
	require.NoError(t, err)
	assert.True(t, match)
}
