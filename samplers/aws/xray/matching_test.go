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

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidArgs(t *testing.T) {
	assert.False(t, wildcardMatch("", "whatever", true))
}

func TestMatchExactPositive(t *testing.T) {
	assert.True(t, wildcardMatch("foo", "foo", true))
}

func TestMatchExactNegative(t *testing.T) {
	assert.False(t, wildcardMatch("foo", "bar", true))
}

func TestSingleWildcardPositive(t *testing.T) {
	assert.True(t, wildcardMatch("fo?", "foo", true))
}

func TestSingleWildcardNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f?o", "boo", true))
}

func TestMultipleWildcardPositive(t *testing.T) {
	assert.True(t, wildcardMatch("?o?", "foo", true))
}

func TestMultipleWildcardNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f??", "boo", true))
}

func TestGlobPositive(t *testing.T) {
	assert.True(t, wildcardMatch("*oo", "foo", true))
}

func TestGlobPositiveZeroOrMore(t *testing.T) {
	assert.True(t, wildcardMatch("foo*", "foo", true))
}

func TestGlobNegativeZeroOrMore(t *testing.T) {
	assert.False(t, wildcardMatch("foo*", "fo0", true))
}

func TestGlobNegative(t *testing.T) {
	assert.False(t, wildcardMatch("fo*", "boo", true))
}

func TestGlobAndSinglePositive(t *testing.T) {
	assert.True(t, wildcardMatch("*o?", "foo", true))
}

func TestGlobAndSingleNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f?*", "boo", true))
}

func TestPureWildcard(t *testing.T) {
	assert.True(t, wildcardMatch("*", "boo", true))
}

func TestMisc(t *testing.T) {
	animal1 := "?at"
	animal2 := "?o?se"
	animal3 := "*s"

	vehicle1 := "J*"
	vehicle2 := "????"

	assert.True(t, wildcardMatch(animal1, "bat", true))
	assert.True(t, wildcardMatch(animal1, "cat", true))
	assert.True(t, wildcardMatch(animal2, "horse", true))
	assert.True(t, wildcardMatch(animal2, "mouse", true))
	assert.True(t, wildcardMatch(animal3, "dogs", true))
	assert.True(t, wildcardMatch(animal3, "horses", true))

	assert.True(t, wildcardMatch(vehicle1, "Jeep", true))
	assert.True(t, wildcardMatch(vehicle2, "ford", true))
	assert.False(t, wildcardMatch(vehicle2, "chevy", true))
	assert.True(t, wildcardMatch("*", "cAr", true))

	assert.True(t, wildcardMatch("*/foo", "/bar/foo", true))
}

func TestCaseInsensitivity(t *testing.T) {
	assert.True(t, wildcardMatch("Foo", "Foo", false))
	assert.True(t, wildcardMatch("Foo", "Foo", true))

	assert.False(t, wildcardMatch("Foo", "FOO", false))
	assert.True(t, wildcardMatch("Foo", "FOO", true))

	assert.True(t, wildcardMatch("Fo*", "Foo0", false))
	assert.True(t, wildcardMatch("Fo*", "Foo0", true))

	assert.False(t, wildcardMatch("Fo*", "FOo0", false))
	assert.True(t, wildcardMatch("Fo*", "FOO0", true))

	assert.True(t, wildcardMatch("Fo?", "Foo", false))
	assert.True(t, wildcardMatch("Fo?", "Foo", true))

	assert.False(t, wildcardMatch("Fo?", "FOo", false))
	assert.True(t, wildcardMatch("Fo?", "FoO", false))
	assert.True(t, wildcardMatch("Fo?", "FOO", true))
}

func TestLongStrings(t *testing.T) {
	chars := []byte{'a', 'b', 'c', 'd'}
	text := bytes.NewBufferString("a")
	for i := 0; i < 8192; i++ {
		text.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	text.WriteString("b")

	assert.True(t, wildcardMatch("a*b", text.String(), true))
}

func TestNoGlobs(t *testing.T) {
	assert.False(t, wildcardMatch("abcd", "abc", true))
}

func TestEdgeCaseGlobs(t *testing.T) {
	assert.True(t, wildcardMatch("", "", true))
	assert.True(t, wildcardMatch("a", "a", true))
	assert.True(t, wildcardMatch("*a", "a", true))
	assert.True(t, wildcardMatch("*a", "ba", true))
	assert.True(t, wildcardMatch("a*", "a", true))
	assert.True(t, wildcardMatch("a*", "ab", true))
	assert.True(t, wildcardMatch("a*a", "aa", true))
	assert.True(t, wildcardMatch("a*a", "aba", true))
	assert.True(t, wildcardMatch("a*a", "aaa", true))
	assert.True(t, wildcardMatch("a*a*", "aa", true))
	assert.True(t, wildcardMatch("a*a*", "aba", true))
	assert.True(t, wildcardMatch("a*a*", "aaa", true))
	assert.True(t, wildcardMatch("a*a*", "aaaaaaaaaaaaaaaaaaaaaaa", true))
	assert.True(t, wildcardMatch("a*b*a*b*a*b*a*b*a*",
		"akljd9gsdfbkjhaabajkhbbyiaahkjbjhbuykjakjhabkjhbabjhkaabbabbaaakljdfsjklababkjbsdabab", true))
	assert.False(t, wildcardMatch("a*na*ha", "anananahahanahana", true))
}

func TestMultiGlobs(t *testing.T) {
	assert.True(t, wildcardMatch("*a", "a", true))
	assert.True(t, wildcardMatch("**a", "a", true))
	assert.True(t, wildcardMatch("***a", "a", true))
	assert.True(t, wildcardMatch("**a*", "a", true))
	assert.True(t, wildcardMatch("**a**", "a", true))

	assert.True(t, wildcardMatch("a**b", "ab", true))
	assert.True(t, wildcardMatch("a**b", "abb", true))

	assert.True(t, wildcardMatch("*?", "a", true))
	assert.True(t, wildcardMatch("*?", "aa", true))
	assert.True(t, wildcardMatch("*??", "aa", true))
	assert.False(t, wildcardMatch("*???", "aa", true))
	assert.True(t, wildcardMatch("*?", "aaa", true))

	assert.True(t, wildcardMatch("?", "a", true))
	assert.False(t, wildcardMatch("??", "a", true))

	assert.True(t, wildcardMatch("?*", "a", true))
	assert.True(t, wildcardMatch("*?", "a", true))
	assert.False(t, wildcardMatch("?*?", "a", true))
	assert.True(t, wildcardMatch("?*?", "aa", true))
	assert.True(t, wildcardMatch("*?*", "a", true))

	assert.False(t, wildcardMatch("*?*a", "a", true))
	assert.True(t, wildcardMatch("*?*a*", "ba", true))
}
