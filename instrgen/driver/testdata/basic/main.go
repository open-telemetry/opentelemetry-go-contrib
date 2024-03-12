// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrgen/rtlib"
)

func recur(n int) {

	if n > 0 {
		recur(n - 1)
	}
}

func main() {

	rtlib.AutotelEntryPoint()
	fmt.Println(FibonacciHelper(10))
	recur(5)
	goroutines()
	pack()
	methods()
}
