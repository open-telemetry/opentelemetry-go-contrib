// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux
// +build linux

package host // import "go.opentelemetry.io/contrib/detectors/host"

import (
	"os"
	"strings"
)

func getHostId() (string, error) {
	machineId, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return "", err
	}

	return strings.Trim(string(machineId), "\n"), nil
}
