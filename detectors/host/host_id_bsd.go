// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build dragonfly || freebsd || netbsd || openbsd || solaris
// +build dragonfly freebsd netbsd openbsd solaris

package host // import "go.opentelemetry.io/contrib/detectors/host"

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func getHostId() (string, error) {
	machineId, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		return strings.Trim(string(machineId), "\n"), nil
	}

	machineId, err = exec.Command("kenv", "-q", "smbios.system.uuid").Output()
	if err == nil {
		return strings.Trim(string(machineId), "\n"), nil
	}

	return "", errors.New("host id not found in: /etc/hostid or kenv")
}
