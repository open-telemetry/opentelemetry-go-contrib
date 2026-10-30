// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package x contains support for otelhttp experimental features.
//
// This package should only be used for features defined in the specification.
// It should not be used for experiments or new project ideas.
package x

import (
	"os"
	"slices"
	"strings"
)

// OptInMetrics is an experimental feature flag that defines which opt-in
// HTTP metrics should be produced. It stands in for the opt-in metric
// advisory parameter until the metrics API supports it.
//
// To enable opt-in metrics set the OTEL_GO_X_HTTP_METRICS_OPTIN environment
// variable to a comma-separated list of metric names (e.g.
// "http.client.active_requests").
var OptInMetrics = newListFeature("HTTP_METRICS_OPTIN")

// ListFeature is an experimental feature control flag that holds a
// comma-separated list of enabled values.
type ListFeature struct {
	key string
}

func newListFeature(suffix string) ListFeature {
	const envKeyRoot = "OTEL_GO_X_"
	return ListFeature{key: envKeyRoot + suffix}
}

// Key returns the environment variable key that needs to be set to enable the
// feature.
func (f ListFeature) Key() string { return f.key }

// Enabled returns if value is listed in the feature.
func (f ListFeature) Enabled(value string) bool {
	return slices.ContainsFunc(strings.Split(os.Getenv(f.key), ","), func(v string) bool {
		return strings.TrimSpace(v) == value
	})
}
