// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestDeprecatedAttributeKeys(t *testing.T) {
	tests := []struct {
		name string
		got  attribute.Key
		want attribute.Key
	}{
		{name: "read bytes", got: ReadBytesKey, want: attribute.Key("http.read_bytes")},
		{name: "read error", got: ReadErrorKey, want: attribute.Key("http.read_error")},
		{name: "wrote bytes", got: WroteBytesKey, want: attribute.Key("http.wrote_bytes")},
		{name: "write error", got: WriteErrorKey, want: attribute.Key("http.write_error")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("attribute key = %q, want %q", test.got, test.want)
			}
		})
	}
}
