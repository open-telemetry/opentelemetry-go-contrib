// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpc_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"
)

var versionRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
	`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func TestVersionSemver(t *testing.T) {
	v := vpc.Version()
	assert.NotNil(t, versionRegex.FindStringSubmatch(v), "version is not semver: %s", v)
}

func TestSemVersion(t *testing.T) {
	assert.Equal(t, vpc.Version(), vpc.SemVersion())
}
