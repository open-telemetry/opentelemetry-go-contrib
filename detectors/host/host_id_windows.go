// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows
// +build windows

package host // import "go.opentelemetry.io/contrib/detectors/host"

import (
	"golang.org/x/sys/windows/registry"
)

func getHostId() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}

	defer key.Close()

	machineId, _, err := key.GetStringValue("MachineGuid")

	return machineId, err
}
