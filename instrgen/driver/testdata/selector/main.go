// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	"go.opentelemetry.io/contrib/instrgen/rtlib"
)

type Driver interface {
	Foo(i int)
}

type Impl struct {
}

func (impl Impl) Foo(i int) {

}

func main() {

	rtlib.AutotelEntryPoint()
	a := []Driver{
		Impl{},
	}
	var d Driver
	d = Impl{}
	d.Foo(3)
	a[0].Foo(4)
}
