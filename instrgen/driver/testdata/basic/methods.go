// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
