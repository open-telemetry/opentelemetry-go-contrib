// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wildcard

import "testing"

func TestMatcher(t *testing.T) {
	tests := []struct {
		name     string
		included []string
		excluded []string
		value    string
		want     bool
	}{
		{
			name:  "no patterns includes all",
			value: "service.name",
			want:  true,
		},
		{
			name:     "literal match",
			included: []string{"service.name"},
			value:    "service.name",
			want:     true,
		},
		{
			name:     "literal mismatch",
			included: []string{"service.name"},
			value:    "service.version",
		},
		{
			name:     "case sensitive",
			included: []string{"service.*"},
			value:    "Service.name",
		},
		{
			name:     "asterisk matches empty",
			included: []string{"service.*"},
			value:    "service.",
			want:     true,
		},
		{
			name:     "asterisk matches all",
			included: []string{"*"},
			want:     true,
		},
		{
			name:     "asterisk matches characters",
			included: []string{"service.*"},
			value:    "service.instance.id",
			want:     true,
		},
		{
			name:     "asterisk matches slash",
			included: []string{"service*id"},
			value:    "service/instance/id",
			want:     true,
		},
		{
			name:     "question mark matches one character",
			included: []string{"process.?ommand_args"},
			value:    "process.command_args",
			want:     true,
		},
		{
			name:     "question mark does not match empty",
			included: []string{"process.?ommand_args"},
			value:    "process.ommand_args",
		},
		{
			name:     "question mark does not match two characters",
			included: []string{"process.?ommand_args"},
			value:    "process.xxommand_args",
		},
		{
			name:     "question mark matches one unicode character",
			included: []string{"service.?"},
			value:    "service.é",
			want:     true,
		},
		{
			name:     "patterns use logical OR",
			included: []string{"host.*", "service.*"},
			value:    "service.name",
			want:     true,
		},
		{
			name:     "multiple asterisks",
			included: []string{"*service**name*"},
			value:    "my.service.name.suffix",
			want:     true,
		},
		{
			name:     "brackets are literals",
			included: []string{"service.[name]"},
			value:    "service.[name]",
			want:     true,
		},
		{
			name:     "brackets do not form a character class",
			included: []string{"service.[name]"},
			value:    "service.n",
		},
		{
			name:     "backslash is literal",
			included: []string{`service.\*`},
			value:    `service.\name`,
			want:     true,
		},
		{
			name:     "exclusion only",
			excluded: []string{"secret.*"},
			value:    "secret.token",
		},
		{
			name:     "exclusion does not match",
			excluded: []string{"secret.*"},
			value:    "service.name",
			want:     true,
		},
		{
			name:     "literal exclusion takes precedence",
			included: []string{"service.name"},
			excluded: []string{"service.name"},
			value:    "service.name",
		},
		{
			name:     "wildcard exclusion takes precedence",
			included: []string{"service.*"},
			excluded: []string{"service.secret*"},
			value:    "service.secret.token",
		},
		{
			name:     "exclude all",
			excluded: []string{"*"},
			value:    "service.name",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher := NewMatcher(test.included, test.excluded)
			if got := matcher.Match(test.value); got != test.want {
				t.Fatalf("Match(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
