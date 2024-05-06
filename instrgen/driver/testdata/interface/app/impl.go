// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package app

import (
	"fmt"
)

type BasicSerializer struct {
}

func (b BasicSerializer) Serialize() {

	fmt.Println("Serialize")
}
