// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerremote

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestClearTraceStateTh(t *testing.T) {
	const rv = "rv:0123456789abcd"

	tests := []struct {
		name string
		// "ot" entry values; empty means no entry.
		in   string
		want string
	}{
		{name: "no ot entry"},
		{name: "th is the only sub-key", in: "th:8"},
		{name: "th first", in: "th:8;" + rv, want: rv},
		{name: "th last", in: rv + ";th:8", want: rv},
		{name: "th in the middle", in: rv + ";th:8;foo:bar", want: rv + ";foo:bar"},
		{name: "no th", in: rv, want: rv},
		{name: "sub-key containing th", in: "path:8", want: "path:8"},
		{name: "trailing semicolon", in: "th:8;"},
		{name: "duplicate th", in: "th:8;" + rv + ";th:9", want: rv},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := oteltrace.TraceState{}
			if tt.in != "" {
				var err error
				ts, err = ts.Insert(traceStateKey, tt.in)
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, clearTraceStateTh(ts).Get(traceStateKey))
		})
	}
}

func TestClearTraceStateThKeepsOtherEntries(t *testing.T) {
	ts, err := oteltrace.TraceState{}.Insert("vendor", "value")
	require.NoError(t, err)
	ts, err = ts.Insert(traceStateKey, "th:8")
	require.NoError(t, err)

	cleared := clearTraceStateTh(ts)
	assert.Empty(t, cleared.Get(traceStateKey))
	assert.Equal(t, "value", cleared.Get("vendor"))
}
