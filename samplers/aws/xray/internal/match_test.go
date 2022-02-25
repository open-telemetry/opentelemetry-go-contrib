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
)

func TestInvalidArgs(t *testing.T) {
	assert.False(t, wildcardMatch("", "whatever"))
}

func TestInvalidArgs1(t *testing.T) {
	assert.True(t, wildcardMatch("*", ""))
}

func TestMatchExactPositive(t *testing.T) {
	assert.True(t, wildcardMatch("foo", "foo"))
}

func TestMatchExactNegative(t *testing.T) {
	assert.False(t, wildcardMatch("foo", "bar"))
}

func TestSingleWildcardPositive(t *testing.T) {
	assert.True(t, wildcardMatch("fo?", "foo"))
}

func TestSingleWildcardNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f?o", "boo"))
}

func TestMultipleWildcardPositive(t *testing.T) {
	assert.True(t, wildcardMatch("?o?", "foo"))
}

func TestMultipleWildcardNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f??", "boo"))
}

func TestGlobPositive(t *testing.T) {
	assert.True(t, wildcardMatch("*oo", "foo"))
}

func TestGlobPositiveZeroOrMore(t *testing.T) {
	assert.True(t, wildcardMatch("foo*", "foo"))
}

func TestGlobNegativeZeroOrMore(t *testing.T) {
	assert.False(t, wildcardMatch("foo*", "fo0"))
}

func TestGlobNegative(t *testing.T) {
	assert.False(t, wildcardMatch("fo*", "boo"))
}

func TestGlobAndSinglePositive(t *testing.T) {
	assert.True(t, wildcardMatch("*o?", "foo"))
}

func TestGlobAndSingleNegative(t *testing.T) {
	assert.False(t, wildcardMatch("f?*", "boo"))
}

func TestPureWildcard(t *testing.T) {
	assert.True(t, wildcardMatch("*", "boo"))
}

func TestMisc(t *testing.T) {
	animal1 := "?at"
	animal2 := "?o?se"
	animal3 := "*s"

	vehicle1 := "J*"
	vehicle2 := "????"

	assert.True(t, wildcardMatch(animal1, "bat"))
	assert.True(t, wildcardMatch(animal1, "cat"))
	assert.True(t, wildcardMatch(animal2, "horse"))
	assert.True(t, wildcardMatch(animal2, "mouse"))
	assert.True(t, wildcardMatch(animal3, "dogs"))
	assert.True(t, wildcardMatch(animal3, "horses"))

	assert.True(t, wildcardMatch(vehicle1, "Jeep"))
	assert.True(t, wildcardMatch(vehicle2, "ford"))
	assert.False(t, wildcardMatch(vehicle2, "chevy"))
	assert.True(t, wildcardMatch("*", "cAr"))

	assert.True(t, wildcardMatch("*/foo", "/bar/foo"))
}

func TestCaseInsensitivity(t *testing.T) {
	assert.True(t, wildcardMatch("Foo", "Foo"))
	assert.True(t, wildcardMatch("Foo", "FOO"))
	assert.True(t, wildcardMatch("Fo*", "Foo0"))
	assert.True(t, wildcardMatch("Fo*", "FOO0"))
	assert.True(t, wildcardMatch("Fo?", "Foo"))
	assert.True(t, wildcardMatch("Fo?", "FOo"))
	assert.True(t, wildcardMatch("Fo?", "FoO"))
	assert.True(t, wildcardMatch("Fo?", "FOO"))
}

func TestLongStrings(t *testing.T) {
	chars := []byte{'a', 'b', 'c', 'd'}
	text := bytes.NewBufferString("a")
	for i := 0; i < 8192; i++ {
		text.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	text.WriteString("b")

	assert.True(t, wildcardMatch("a*b", text.String()))
}

func TestNoGlobs(t *testing.T) {
	assert.False(t, wildcardMatch("abcd", "abc"))
}

func TestEdgeCaseGlobs(t *testing.T) {
	assert.True(t, wildcardMatch("", ""))
	assert.True(t, wildcardMatch("a", "a"))
	assert.True(t, wildcardMatch("*a", "a"))
	assert.True(t, wildcardMatch("*a", "ba"))
	assert.True(t, wildcardMatch("a*", "a"))
	assert.True(t, wildcardMatch("a*", "ab"))
	assert.True(t, wildcardMatch("a*a", "aa"))
	assert.True(t, wildcardMatch("a*a", "aba"))
	assert.True(t, wildcardMatch("a*a", "aaa"))
	assert.True(t, wildcardMatch("a*a*", "aa"))
	assert.True(t, wildcardMatch("a*a*", "aba"))
	assert.True(t, wildcardMatch("a*a*", "aaa"))
	assert.True(t, wildcardMatch("a*a*", "aaaaaaaaaaaaaaaaaaaaaaa"))
	assert.True(t, wildcardMatch("a*b*a*b*a*b*a*b*a*",
		"akljd9gsdfbkjhaabajkhbbyiaahkjbjhbuykjakjhabkjhbabjhkaabbabbaaakljdfsjklababkjbsdabab"))
	assert.False(t, wildcardMatch("a*na*ha", "anananahahanahana"))
}

func TestMultiGlobs(t *testing.T) {
	assert.True(t, wildcardMatch("*a", "a"))
	assert.True(t, wildcardMatch("**a", "a"))
	assert.True(t, wildcardMatch("***a", "a"))
	assert.True(t, wildcardMatch("**a*", "a"))
	assert.True(t, wildcardMatch("**a**", "a"))

	assert.True(t, wildcardMatch("a**b", "ab"))
	assert.True(t, wildcardMatch("a**b", "abb"))

	assert.True(t, wildcardMatch("*?", "a"))
	assert.True(t, wildcardMatch("*?", "aa"))
	assert.True(t, wildcardMatch("*??", "aa"))
	assert.False(t, wildcardMatch("*???", "aa"))
	assert.True(t, wildcardMatch("*?", "aaa"))

	assert.True(t, wildcardMatch("?", "a"))
	assert.False(t, wildcardMatch("??", "a"))

	assert.True(t, wildcardMatch("?*", "a"))
	assert.True(t, wildcardMatch("*?", "a"))
	assert.False(t, wildcardMatch("?*?", "a"))
	assert.True(t, wildcardMatch("?*?", "aa"))
	assert.True(t, wildcardMatch("*?*", "a"))

	assert.False(t, wildcardMatch("*?*a", "a"))
	assert.True(t, wildcardMatch("*?*a*", "ba"))
}