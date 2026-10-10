// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_azure_vm

package detectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureVMDetector(t *testing.T) {
	assert.NotNil(t, azurevmDetector())
}
