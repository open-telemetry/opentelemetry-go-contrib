// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"os"
)

func Close() error {
	return nil
}

func pack() {

	f, e := os.Create("temp")
	defer f.Close()
	if e != nil {

	}
}
