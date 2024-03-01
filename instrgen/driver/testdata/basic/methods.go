// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

type element struct {
}

type driver struct {
	e element
}

type i interface {
	anotherfoo(p int) int
}

type impl struct {
}

func (i impl) anotherfoo(p int) int {

	return 5
}

func anotherfoo(p int) int {
	return 1
}

func (d driver) process(a int) {

}

func (e element) get(a int) {

}

func methods() {

	d := driver{}
	d.process(10)
	d.e.get(5)
	var in i
	in = impl{}
	in.anotherfoo(10)
}
