// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	assert.Empty(t, Options(Detector{}))
	assert.Len(t, Options(Detector{
		AWSEC2:  true,
		AWSECS:  true,
		AWSEKS:  true,
		AzureVM: true,
		GCP:     true,
	}), 5)
}
