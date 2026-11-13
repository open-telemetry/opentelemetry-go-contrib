// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptInMetrics(t *testing.T) {
	const key = "OTEL_GO_X_HTTP_METRICS_OPTIN"
	require.Equal(t, key, OptInMetrics.Key())

	for _, tc := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "listed", value: "http.client.active_requests", want: true},
		{name: "spaces", value: " other , http.client.active_requests ", want: true},
		{name: "unlisted", value: "other", want: false},
		{name: "prefix", value: "http.client", want: false},
		{name: "empty", value: "", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(key, tc.value)
			assert.Equal(t, tc.want, OptInMetrics.Enabled("http.client.active_requests"))
		})
	}
}
