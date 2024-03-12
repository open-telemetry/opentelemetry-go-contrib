// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
