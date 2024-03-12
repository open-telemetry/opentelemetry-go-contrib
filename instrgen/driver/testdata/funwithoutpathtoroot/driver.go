// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"go.opentelemetry.io/contrib/instrgen/rtlib"
)

func bar() {

}

func foo() {
	bar()
}

func main() {
	rtlib.AutotelEntryPoint()
	bar()
}
