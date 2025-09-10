// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package skywalking

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	version := Version()
	assert.NotEmpty(t, version)

	// Version should follow semantic versioning
	semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+`)
	assert.True(t, semverRegex.MatchString(version), "version should follow semantic versioning")
}
