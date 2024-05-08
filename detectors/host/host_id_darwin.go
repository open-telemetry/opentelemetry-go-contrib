// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin
// +build darwin

package host // import "go.opentelemetry.io/contrib/detectors/host"

import (
	"os/exec"
	"strings"
)

func getHostId() (string, error) {
	machineId, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", err
	}

	return strings.Trim(string(machineId), "\n"), nil
}
