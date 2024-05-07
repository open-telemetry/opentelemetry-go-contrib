// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"fmt"
)

func goroutines() {

	messages := make(chan string)

	go func() {

		messages <- "ping"
	}()

	msg := <-messages
	fmt.Println(msg)

}
